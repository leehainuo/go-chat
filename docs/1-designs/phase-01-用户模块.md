# Design Card｜用户模块

- 日期：2026-10-27
- 作者：leehainuo
- 相关需求 / PR / 文档：tmp/Infinitechat-Go/Infinitechat-Go项目课件.md（课件）；实施计划 tmp/phase-01-用户模块.md
- 风险等级：中（原因：账号唯一性、缓存一致性、事务）
- 设计版本：v1

## 1. 我要解决什么问题？

- **业务目标**：
    - 实现用户的登录注册与退出
    - 实现用户查看自己的信息
    - 实现用户更新和上传头像
    - 实现初始化余额和查看余额流水明细
- **非目标**：
    - 真实短信发送
    - 用户注销功能
- **验收场景**：
    - Given 未注册手机号或邮箱且密码正确 When 注册 Then 数据库落数据、返回 accessToken + refreshToken + wsServerUri
    - Given 已注册用户、正确密码 When 登录 Then 返回 accessToken + refreshToken + wsServerUri 且将该账号旧 refreshToken 删除
    - Given 已登录用户 When 改密码成功 Then 旧密码不能再登录
    - Given 已登录用户 When 用预签名 URL 直传 MinIO 后调 /update/avatar Then getUserInfo 返回新头像。
- **关键不变量**：
    1. 一个手机号或邮箱全局只能绑定一个账号
    2. 注册成功的 创建用户、初始化余额同事务, 要么全成要么全部回滚
    3. 同一 userID 在任意时刻最多只能有一个有效 refreshToken
    4. 余额都以'分'存储, 转换为'元'显示

## 2. 已知事实、约束和未知项

- **已知事实与证据来源**：
    - 采用 go-zero 框架进行项目开发, 用户模块单独为 user-api :8001

- **约束**：
    - 本地单机, 使用 docker-compose.yaml; user-api :8001 固定
    - 密码加密方案采用 MD5(salt + password)

- **假设 / 待确认**：

## 3. 我的初版技术设计（重点由自己填写）

- **核心流程 / 状态变化**：
    1. 验证码(Redis SETEX 300s, 重复发送拒绝) 
    2. 注册 校验手机号和邮箱合法性 -> 事务写用户信息和余额 -> 签发 Token 和 wsServerUri
    3. 登录 校验手机号或邮箱是否存在 -> 校验密码是否正确 -> 签发 Token 和 wsServerUri 并且踢旧会话 -> 无感续签
    4. 退出 删除 refreshToken 和 wsServerUri
- **API 契约 (prefix /api/user)**：
    - `GET  /sendCaptcha            验证码`
    - `POST /loginCode              验证码登录`
    - `POST /loginPwd               密码登录`
    - `POST /register               注册`
    - `POST /logout                 登出`
    - `POST /info                   查看用户信息`
    - `POST /updatePwd              更新密码`
    - `POST /update/avatar          更新头像`
    - `GET  /refresh/token          续签`
    - `GET  /refresh/wsServerUri    重写分配节点`
    - `GET  /uploadUrl              MinIO 预签名 PUT URL`
    - `GET  /balance/:userID        获取当前用户余额`
    - `GET  /balanceDetail/:userID  获取当前用户余额流水明细`

- **数据模型与访问模式**：
    1. user 表: userID 主键唯一索引; email、phone 唯一可空; avatar 存储 url
    2. user_balance 表: 1:1 userID 和 balanceID 一对一; 并有一个总余额字段
    3. balance_log 表

- **事务 / 并发 / 幂等策略**：
    - 并发注册 依靠数据库 唯一索引 天然防止
    - 事务使用 go-zero 自带的 sqlx 的事务能力
    - 幂等方面 验证码使用 redis 进行缓存, 防止刷验证码
    - 单点 redis 写 user:refresh:{userID} 需先删除 old user:refresh:{userID}

- **权限 / 失败处理 / 重试策略**：
    - 错误码统一使用标准库 net/http 自带的
    - 失败处理统一符合 Go 社区理念 显示 err 严禁 panic

- **与现有实现如何衔接**：
    - 除了在 user-api 模块会被使用之外, 其它模块也需要的, 都统一到 pkg

## 4. 我自己的推理

- **为什么选这个方案？**：
    - accessToken 和 refreshToken 方案可以实现无感续签和自动踢号, 提升用户体验, accessToken 时间短可以缩小泄漏窗口
    - 一致性哈希 wsServerUri 方案 解决 固定取模 hash 的节点扩容 / 缩容时，大量连接打散、重连
- **考虑过的另一种方案及不采用的原因**：
    - 单 Token 方案 存在泄漏窗口 无法踢下线、无法续签
    - 固定取模 hash 的节点扩容 / 缩容时, 会将大量连接打散、重连
- **我最担心的一种失败方式 / 反例**：
    - 改密码后已签发的 accessToken 在 ≤30min 内仍有效
- **我最没把握的一项判断**：


## 5. 如何证明设计可行？

- **正常场景测试**：
    1. 全链路正常走通
    2. 验证码 300s 后正常可用
    3. 续签时, old user:refresh:{userID} 正常删除, 换发正常成功
- **异常/边界场景测试**：
    1. 过期、错误验证码
    2. 验证码登录不存在账号
- **涉及并发或数据迁移时的额外验证**：
    1. 并发注册 有且只能有一条成功
- **发布、监控、回滚关注点**（如适用）：
    - 关注 Redis key 均带 TTL、MinIO 桶 gochat 需预创建、Kafka topic 自动创建是否生效。

---

## 6. 评审与修订索引（评审后再补）

- Designer 评审纪要：`2-reviews/phase-01-用户模块.md`
- 决定接受的修改：
- 保留原方案的理由与证据：
- 需要负责人确认的事项：
- 修订版本与复评结论：