-- phase-02 联系人模块增量迁移（仅针对已按 phase-01 gochat.sql 初始化的存量库；
-- 全新安装直接执行 deploy/sql/gochat.sql，无需本脚本）
USE `gochat`;

-- 好友详情需要账号状态校验（封禁/注销）与性别、个性签名展示
ALTER TABLE `user`
  ADD COLUMN `state` tinyint NOT NULL DEFAULT '0' COMMENT '账号状态 0正常 1封禁 2注销' AFTER `avatar`,
  ADD COLUMN `gender` tinyint NOT NULL DEFAULT '0' COMMENT '性别 0未知 1男 2女' AFTER `state`,
  ADD COLUMN `description` varchar(255) NOT NULL DEFAULT '' COMMENT '个性签名' AFTER `gender`;

-- 好友关系表（双向两行）
CREATE TABLE `friend` (
  `id` bigint NOT NULL COMMENT '雪花ID',
  `user_id` bigint NOT NULL COMMENT '归属用户ID',
  `friend_id` bigint NOT NULL COMMENT '对方用户ID',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0正常 1拉黑 2删除（保留枚举，物理删除方案下无写入路径）',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_friend` (`user_id`, `friend_id`),
  KEY `idx_friend_user` (`friend_id`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='好友关系表';

-- 好友申请表
CREATE TABLE `apply_friend` (
  `id` bigint NOT NULL COMMENT '雪花ID（申请ID）',
  `sender_id` bigint NOT NULL COMMENT '发送者用户ID',
  `receiver_id` bigint NOT NULL COMMENT '接收者用户ID',
  `message` varchar(255) NOT NULL DEFAULT '' COMMENT '申请留言',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0未读 1通过 2拒绝 3已读 4过期',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_sender_receiver` (`sender_id`, `receiver_id`),
  KEY `idx_receiver_status` (`receiver_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='好友申请表';

-- 会话表
CREATE TABLE `session` (
  `id` bigint NOT NULL COMMENT '雪花ID（会话ID）',
  `type` tinyint NOT NULL DEFAULT '0' COMMENT '0单聊 1群聊 2机器人',
  `name` varchar(64) NOT NULL DEFAULT '' COMMENT '会话名称（单聊为空，展示时取对端昵称）',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='会话表';

-- 用户-会话关联表
CREATE TABLE `user_session` (
  `id` bigint NOT NULL COMMENT '雪花ID',
  `session_id` bigint NOT NULL COMMENT '会话ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session_user` (`session_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户会话关联表';
