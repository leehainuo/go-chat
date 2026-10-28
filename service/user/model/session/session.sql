-- 会话表（type: 0 单聊 / 1 群聊 / 2 机器人通知，本阶段仅产生单聊）
CREATE TABLE `session` (
  `id` bigint NOT NULL COMMENT '雪花ID（会话ID）',
  `type` tinyint NOT NULL DEFAULT '0' COMMENT '0单聊 1群聊 2机器人',
  `name` varchar(64) NOT NULL DEFAULT '' COMMENT '会话名称（单聊为空，展示时取对端昵称）',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='会话表';
