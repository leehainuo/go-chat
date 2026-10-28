# 设计评审：phase-02 联系人模块（v1）

## ① 依据与证据范围

| 类别 | 内容 |
|---|---|
| 评审对象 | `docs/1-designs/phase-02-联系人模块.md`（设计卡 v1，2026-10-28） |
| 需求依据 | `tmp/phase-02-联系人模块.md`（接口清单/核心实现/验收标准）；课件《搜索用户》~《获取好友详情》章节（4969–6213 行） |
| 仓库可证事实 | ① `deploy/sql/gochat.sql` 目前仅有 user / user_balance / balance_log 三表，**friend / apply_friend / session / user_session 均不存在**，且 user 表**没有 state / gender / description 列**；② `JwtAuthMiddleware` 鉴权后已将 userId 注入 context；③ phase-01 注册事务已确立 `TransactCtx + Tx 前缀 model 方法` 的事务模式（`register_logic.go:96-107`）；④ phase-01 注册事务只有三表（用户+余额+流水），**未实现"加机器人为好友"** |
| 课件可证事实 | ① 好友表是**双向两行**结构：`FindListByUserId` 用 `where user_id = ?`，每行有独立 `FriendId`（课件 5087 节）；删除好友 `DeleteByBothDirectionsTx` 物理删除双向两行（5880 节）；拉黑校验 `friendRelation.UserId != userId → NoAuthError`，即**行级归属、单向拉黑**（6087 节）。② 搜索是正则分派后**精确等值**查询（`FindOneByPhone/FindOneByEmail`），返回**单个** FriendDetailVO，非列表（4969 节）。③ 申请表存在 `(sender_id, receiver_id)` 唯一索引——goctl 仅依据唯一索引生成 `FindOneBySenderIdReceiverId`（课件模型说明与 5319 节）。④ 通过申请的课件实现是"查→内存判断→整行 Update"，**无条件更新、无事务包裹**（6131 节）。⑤ 通知与过期事件均为 goroutine fire-and-forget（5395 节） |
| 无法访问 | 课件外部附件 `infiniteChat.sql`（表 DDL 不在仓库），friend/session/user_session 的索引与唯一约束只能从生成的 model 方法名推断 |
| 引用瑕疵 | 设计卡第 5 行将 `tmp/phase-02-联系人模块.md` 称为"实施计划"，其自我定位是需求+实现方案混合文档，本评审将其视为需求依据 |

## ② 方案复述

目标：user-api :8001 内完成 `/api/contact` 10 个接口，覆盖好友关系全生命周期（搜索→申请→通过/拒绝→列表/详情→拉黑/删除），核心链路是**好友申请事务与 Kafka 通知**。三个不变量：申请幂等不重复累计、申请状态并发保护且终态不可改、状态修改与加好友同事务。核心数据模型取舍：好友表两 userID 比较大小后**存放同一列**（单行方案），理由是省内存、强关联、天然一致。验收场景覆盖了 10 个接口的主线，事务意识与 Kafka 风险意识（风险等级行）是亮点。

## ③ 结论：需重新设计（范围限定）

接口清单、事务意识、验收骨架可以保留；但**核心数据模型（好友表单行 Hash 方案）与需求文档、课件全部冲突**，删除好友语义在设计卡内部与需求文档之间三处互相矛盾，必须先定方案再进入实现。

### 阻塞问题

**DR-1｜阻塞｜数据模型（第 73、89 行）｜好友表"单行 Hash 方案"与需求文档/课件的两行双向方案直接冲突，且无法表达单向拉黑**

- 触发条件：按第 73 行实现 friend 表（一行存一对好友，status 挂在行上）。
- 后果：①课件与需求文档（4.6 `DeleteByBothDirectionsTx`、4.7 `UpdateStatusByUserIdFriendId`、`friendRelation.UserId == userId` 归属校验）全部按"每行 (user_id, friend_id)"写，单行方案下这些方法无一能落地；②拉黑是**单向**行为：A 拉黑 B 后，两行方案里 A 行=Blocked、B 行=Normal，B 完全无感知（微信语义）；单行方案只有一份 status，A 拉黑 B 会导致 B 的好友列表也把 A 显示为"拉黑"，且阶段 4 消息校验缓存 `msg:validate:friend:status:{a}:{b}` 是按方向取 key 的，单向语义被破坏；③好友列表查询变为 `WHERE user_a = me OR user_b = me`，OR 条件使两个单列索引都难以命中。
- 依据/待验证：课件 5087/5880/6087 节（见①）；"省内存"收益是每对好友一行 vs 两行，好友关系量级下可忽略；"天然一致性"在两行方案里由事务插入两行同样成立，不构成单行的独占优势。
- 推荐修改：改为**两行双向**（与需求文档一致），`UNIQUE KEY (user_id, friend_id)`；一致性由 phase-01 已确立的事务模式保证。
- 待确认：是否有课件之外的理由坚持单行方案（如刻意练习该模式）？若坚持，需求文档 4.6/4.7 与阶段 4 缓存 key 语义需同步重写。

**DR-2｜阻塞｜删除好友（第 24 行 vs 需求文档 4.6）｜"软删除"与"物理删除四处数据"三处互相矛盾**

- 触发条件：设计卡第 24 行验收写"软删除好友表两者的列数据"，第 59 行流程写"修改好友表字段"（软删）；需求文档 4.6 与课件 5880 节是**物理 DELETE**：apply_friend 双向、friend 双向、user_session、session 四处全删 + 缓存清理。
- 后果：实现时无法判断听谁的；两种语义的产品行为完全不同——物理删会把两人单聊会话一起删掉（历史聊天入口消失），软删则保留。另外约束里的 `Deleted=2` 状态在物理删除下永远不会出现在库里，`FindListByUserId` 的 `status != 2` 过滤成为死代码（课件本身即如此）。
- 推荐修改：二选一并写进设计卡：A=物理删（与课件一致，阶段 4/5 的消息校验按"删除后会话不存在"设计，复刻成本最低）；B=软删 `status=Deleted`（保留会话与记录，更贴近微信，但要重写需求文档 4.6 并处理会话归属）。课程复刻语境下倾向 A，同时把"删除会话导致聊天记录入口消失"记为已接受的产品行为。
- 待确认：选 A 还是 B？`Deleted=2` 与 `status != 2` 过滤的去留随之定。

### 重要问题

**DR-3｜重要｜权限（第 80 行）｜10 条路由全部携带 `:userID` 路径参数，越权（IDOR）风险与 phase-01 DR-3 同款**

- 触发条件：设计卡只写"必须为当前用户已登录状态"，未写"路径 userID 必须与 token 身份一致"。已登录用户 A 调 `DELETE /:B/friend/:C` 即可替 B 删好友。
- 依据：仓库 `JwtAuthMiddleware` 已把 userId 注入 context，logic 直接取 ctx 即可——路径参数 `:userID` 整体可以删掉，比"取路径再比对"更简单也更安全。课件代码直接信任 `req.UserId`，属已知缺陷，不应照抄。
- 推荐修改：联系人全部接口从 ctx 取身份，路由去掉 `:userID`；若因课程对齐保留路径参数，则必须逐接口校验 `ctx.userID == path.userID`（不等返回 NoAuthError）。

**DR-4｜重要｜并发/幂等（第 31-32、74 行）｜不变量 1、2 只写了"要保护"，没写"靠什么保护"**

- 触发条件 1（发申请）：第 74 行只写"给 userID 设置索引"。并发两次发申请同时通过"无待处理申请"检查 → 双插两行。课件有 `(sender_id, receiver_id)` 唯一索引兜底（`FindOneBySenderIdReceiverId` 由唯一索引生成），设计卡未写。同时"被拒/过期后能否重新申请"未定义：若唯一键是 `(sender_id, receiver_id)`，二次申请 Insert 会撞键——课件只在 Unread/Read 时复用行，Rejected/Expired 后新建行的行为与唯一键矛盾，需要明确冲突策略。
- 触发条件 2（通过申请）：课件是"查→内存判断→整行 Update"，两个并发 accept 都通过 `status ∈ {Unread, Read}` 校验 → 都调 AddFriend → friend / session / user_session 重复行。设计卡不变量 2 要求"终态不可改"，但没有给机制。
- 推荐修改：①申请表显式声明 `UNIQUE KEY (sender_id, receiver_id)`，duplicate key 映射为"复用行更新或业务提示"；②accept 改为条件更新 `UPDATE apply_friend SET status=Accepted WHERE apply_friend_id=? AND status IN (0,3)`，`RowsAffected==1` 才继续 AddFriend，且全程置于 txctx 事务内；③friend 表 `UNIQUE (user_id, friend_id)`、user_session 表 `UNIQUE (session_id, user_id)` 兜底；④补一张状态转换矩阵（Unread→Accepted/Rejected/Read，Expired 由过期消费者写，其余 from→to 非法）。
- 待确认：Rejected / Expired 后再次申请的产品规则（允许新建？限频？）。

**DR-5｜重要｜事务与 Kafka 通知顺序（第 6、33 行）｜"事务"与"Kafka 通知"的相对顺序未定义**

- 触发条件：不变量 3 要求状态修改+加好友同事务；课件把 101/102 通知与过期事件放 goroutine fire-and-forget。若先发通知后提交：事务回滚 → 对端收到"申请已通过"但点进去无会话（幽灵通知）；若提交后发送：Kafka 失败 → 对端永远无感知，过期事件丢失则申请永不过期。
- 推荐修改：顺序定为**事务提交成功后 → 推 Kafka/通知**；发送失败只记 Error 日志、不影响主流程（课件语义，如实写进设计卡）；过期事件明确"消费端失败重试 + 定时扫表兜底"或声明归属后续阶段。课程语境不必引入 Outbox，但取舍要落在纸面上。

**DR-6｜重要｜搜索语义与表结构衔接（第 23、53、83 行）｜"右模糊列表"与需求的"精确单条"冲突；DDL 变更清单缺失**

- 触发条件 1：第 23 行验收写"模糊搜索 右模糊 返回匹配的联系人**列表**"，第 53 行流程写"右模糊查询"；需求文档与课件（4969 节）是正则分派后**等值精确**查询（手机/邮箱本就唯一），返回**单个** FriendDetailVO，关键字不匹配正则直接 ParamsError。唯一列上的右模糊最多命中 1 条，"列表"不成立。
- 触发条件 2：`deploy/sql/gochat.sql` 的 user 表没有 `state / gender / description` 列，而好友详情（需求文档 4.1）需要 State 封禁/注销校验和 Gender/Signature 字段；friend/apply_friend/session/user_session 四表也不存在。第 83 行"与现有实现如何衔接"为空。
- 推荐修改：①搜索改为"正则分派 + 精确查 + 单条返回"，或明确升级为昵称右模糊列表（那是另一套索引与隐私设计）；②补 DDL 变更清单：新建四表（含唯一键/索引）+ `ALTER user ADD state/gender/description`；③第 83 行记录跨阶段契约 `msg:validate:friend:status:{a}:{b}` 缓存（阶段 4 依赖，删除/拉黑后 evict）。

### 建议项（不阻塞）

- 第 67-68 行 block 路由的**注释与方法写反**：POST 应为拉黑、DELETE 应为取消拉黑（与需求文档二、节表格相反）。
- 路径参数命名 `receiveuserID / friendID` 与需求文档 `receiveuserId / friendId` 大小写不一致；goctl 区分大小写，写 .api 前统一。
- `POST /application/:status` 名为批量，实际通过只处理 `senderIds[0]`（课件如此）；设计卡应标注该限制，避免前端误传多个 ID。
- 好友列表是 N+1（每个好友 GetUserById + 三表 JOIN 查 sessionId）+ 内存分页；课程规模可接受，建议记为"已知取舍"，或改 IN 批量查用户 + sessionId 批量查。
- 第 39 行未读数口径"查询该用户申请表状态为 Unread"未限定方向；应写明 `receiver_id = 我 AND status = Unread`（课件 CountByReceiverAndStatus），否则自己发出的申请也会被计入。
- 第 19-21 行非目标为空：两件事应显式写进去——①phase-01 裁掉的"注册加机器人为好友"是否在本模块补做（friend 表本次建好）；②申请过期（Expired=4）依赖的定时任务消费者归属哪个阶段。
- `NonFriend=-1` 是 VO 计算值（查不到好友关系时填充），不是落库状态，建议在约束行注明，避免被误建成枚举列。
- 搜索可按手机号枚举探活用户，课程项目可接受；与 phase-01 DR-4 同款，建议在设计卡风险行记录"已接受"。

## ④ 方案比较

**好友表：A（作者，单行 pair-row）vs B（两行双向）**。A 的真实成本：单向拉黑不可表达（DR-1）、列表查询 OR 索引失效、需求文档全部 model 方法重写、与阶段 4 缓存契约冲突；A 的收益：每对好友省一行（可忽略）、无需事务插两行。B 的成本：插入/删除要事务保证两行一致（phase-01 模式现成）、删除要双向 DELETE（课件现成）。**倾向 B**，前提是本项目仍以复刻课件、阶段 4/5 顺滑衔接为约束。若作者刻意想练习 pair-row 模式，可以选 A，但必须同步重写需求文档 4.6/4.7 并放弃单向拉黑语义——那是产品妥协，不只是技术取舍。

**删除：物理删 vs 软删**——见 DR-2，课程复刻倾向物理删，产品完整性倾向软删，二选一即可，不存在必须推翻的问题。

## ⑤ 必须由作者回答的问题（按序）

1. DR-1：好友表改两行双向，还是坚持单行并重写需求文档与单向拉黑语义？
2. DR-2：删除好友选物理删（四处全清）还是软删（保留会话）？`Deleted=2` 去留？
3. DR-3：`:userID` 路径参数删除（改用 ctx 身份），还是保留 + 逐接口比对？
4. DR-4：申请表唯一键与"二次申请"规则（Rejected/Expired 后允许新建还是提示）？accept 并发保护采用条件更新方案吗？
5. DR-5：Kafka 通知定为"提交后发送、失败仅日志"吗？过期任务兜底归属哪个阶段？
6. DR-6：搜索定"精确单条"还是"右模糊列表"？user 表加列的 DDL 变更清单何时补？

**回答:**
DR-1：改两行双向（推荐 B） 需求文档 4.6/4.7 的全部 model 方法、课件实现、阶段 4 的单向缓存契约都按两行写，选两行是零改造成本；单行方案要重写需求文档、牺牲单向拉黑，还要解决 OR 查询索引问题。代价：插入/删除好友要在事务里写两行——而 phase-01 的 TransactCtx 事务模式现成，这个代价几乎为零。

DR-2：物理删（四处全清），Deleted=2 保留常量但注明无写入路径 阶段 4/5 的消息校验按“删除后会话不存在”设计，物理删与后续阶段自洽；软删要重写需求文档 4.6，且保留会话引出“删除好友后会话列表还显示对方”的新产品问题，超出本期范围。代价是产品妥协：删除后聊天记录入口消失——课程复刻语境下可接受，写进设计卡风险行即可。Deleted=2 建议保留（项目 README 和课件常量表都列了它），但注明“当前无写入路径”，status != 2 过滤作为防御条件留着无害。

DR-3：保留 :userID + 逐接口比对 我核实过 phase-01 的实现：balance_detail_logic.go:31-37 已经确立了 mustUserId(ctx) + req.UserId != userId → Forbidden 的模式。联系人模块照抄这个模式成本最低（两行代码、helper 现成），且与课件 URL 形状一致——如果你后面复用课件前端代码，不用改调用方。删路径参数是更干净的终态，但会推翻已上线的 phase-01 契约，不值得在本期做。关键是不管选哪个，漏一个接口没比对就是越权，所以统一封装一个 helper（如 mustSelf(ctx, req.UserId)）强制所有 logic 首行调用。

DR-4：UNIQUE KEY (sender_id, receiver_id)；二次申请改为“复用行”；accept 用条件更新（是） 这里我发现课件有个潜在 bug 值得你知道：课件的 SendFriendRequest 只在 status ∈ {Unread, Read} 时复用行，如果原申请是 Rejected/Expired，会落到 Insert → 撞唯一键 → 返回 SystemError（系统错误）。所以推荐规则改成：存在申请行即复用（UPDATE status=Unread + 新 message + 新 updated_time，无论 Rejected/Expired/Read），既支持“被拒后重新申请”（微信语义），又修掉撞键崩溃。Accepted 的行走不到这里——前面“已是好友”就拦截了，配合 DR-2 物理删也没有残留行。accept 并发保护：条件更新 UPDATE ... WHERE apply_friend_id=? AND status IN (0,3)，RowsAffected==1 才继续 AddFriend，全程在事务内。别忘了 friend 表 UNIQUE (user_id, friend_id)、user_session 表 UNIQUE (session_id, user_id) 兜底。

DR-5：通知定为“提交后发送、失败仅 Error 日志”；过期兜底挂 phase-07 我查了 tmp/README.md 的 topic 总表：friend-request-expiration-topic 至今没有生产者/消费者归属（预留），而 phase-07 红包模块已经规划了完整的“过期调度器”基础设施（Redis ZSet 延迟队列 + 分布式锁 lock:expiration-dispatcher + 扫描过期红包）。推荐：本阶段只承诺“事务提交成功后发 friend-request-creation-topic，失败记日志不回滚”；过期消费端写进非目标；兜底归属 phase-07，复用调度器，加一个不依赖 Kafka 的 SQL 扫描（status IN (0,3) AND updated_time < 24h 前 → Expired）作保底。最晚 phase-10 联调前补齐。代价：24h 内过期不生效（申请多挂一会），无实际危害，风险可接受。

DR-6：精确单条（正则分派 + 等值查询），DDL 清单进实现前补 手机/邮箱是唯一列，右模糊最多命中 1 条，“返回列表”这个语义本身不成立，不用纠结。想要“模糊搜人”是另一个功能（昵称模糊 + 不同索引 + 更大的隐私暴露面），将来想要就单独立项。DDL 清单（friend/apply_friend/session/user_session 四表含唯一键，ALTER user ADD state/gender/description）是数据模型定案的直接产物，应在按 v2 设计卡开工前写进 deploy/sql/gochat.sql 和各 model 的 .sql 文件——它本身就是你第一个编码动作，不存在“以后再补”的时机。



## ⑥ 验证与复评入口

- 正常用例：A 发申请 → B 未读数=1 → B 通过 → friend 表双向两行 + session/user_session 各一行 → 双方好友列表互见 → 任一方删除后 friend/apply_friend/session/user_session 四处清空（按 DR-2 选定语义）。
- 异常用例：①重复发申请：命中唯一键/复用行，返回业务提示而非双插；②对 Accepted 记录再发 accept：返回"该申请不可被处理"；③A 拉黑 B 后：A 行 status=Blocked、B 行仍 Normal、B 侧列表 A 显示正常；④搜索非手机/邮箱关键字：ParamsError。
- 并发用例：10 并发对同一对用户发申请仅落 1 行；两个并发 accept 同一申请，恰一个成功建好友、另一个失败回滚。
- 事务/消息用例：事务中 AddFriend 失败 → apply 状态保持不变（第 109 行验收的直接验证）；提交后 Kafka 发送失败 → 主流程成功 + Error 日志留存。
- 复评：作者修订后对照本清单逐条标注`已解决 / 维持原方案（附证据）/ 交由负责人`，更新设计卡 §6 索引，评审者复评。

**总体判断**：接口面、事务意识、风险意识都到位，问题集中在**数据模型与需求文档的冲突未暴露**（DR-1/DR-2）和**不变量缺机制**（DR-4/DR-5）。DR-1、DR-2 定案、DR-3~DR-6 补齐后即可进入实现；不需要推翻接口设计与整体链路。
