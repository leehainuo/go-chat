-- gochat 数据库初始化脚本（phase-01 用户模块 + phase-02 联系人模块）
-- 全新安装直接整体执行；已有 phase-01 库执行 migrations/phase-02-contact.sql 增量迁移
-- 建库
CREATE DATABASE IF NOT EXISTS `gochat` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `gochat`;

-- 用户表
CREATE TABLE `user` (
  `user_id` bigint NOT NULL COMMENT '雪花ID',
  `nickname` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称',
  `password` varchar(64) NOT NULL COMMENT '密码摘要 MD5(salt+password)',
  `phone` varchar(16) DEFAULT NULL COMMENT '手机号，全局唯一可空',
  `email` varchar(64) DEFAULT NULL COMMENT '邮箱，全局唯一可空',
  `avatar` varchar(512) NOT NULL DEFAULT '' COMMENT '头像URL',
  `state` tinyint NOT NULL DEFAULT '0' COMMENT '账号状态 0正常 1封禁 2注销',
  `gender` tinyint NOT NULL DEFAULT '0' COMMENT '性别 0未知 1男 2女',
  `description` varchar(255) NOT NULL DEFAULT '' COMMENT '个性签名',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `uk_phone` (`phone`),
  UNIQUE KEY `uk_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户表';

-- 用户余额表（余额一律以“分”存储）
CREATE TABLE `user_balance` (
  `balance_id` bigint NOT NULL COMMENT '雪花ID',
  `user_id` bigint NOT NULL COMMENT '用户ID，1:1',
  `balance` bigint NOT NULL DEFAULT '0' COMMENT '总余额，单位：分',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`balance_id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户余额表';

-- 余额流水表
CREATE TABLE `balance_log` (
  `log_id` bigint NOT NULL COMMENT '雪花ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `type` tinyint NOT NULL COMMENT '类型 1注册初始化 2充值 3扣减',
  `amount` bigint NOT NULL COMMENT '变动金额，单位分，正负号表示方向',
  `balance_after` bigint NOT NULL COMMENT '变动后余额，单位分',
  `remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`log_id`),
  KEY `idx_user_create` (`user_id`, `create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='余额流水表';

-- 好友关系表（DR-1 决策：双向两行，每行 (user_id, friend_id) 表示一个方向的观察关系，
-- 拉黑/删除只作用于归属行，天然支持单向语义；一致性由写入方在同一事务内插两行保证）
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

-- 好友申请表（DR-4 决策：uk_sender_receiver 唯一键兜底并发重复申请；
-- 被拒/过期后二次申请复用原行改状态回 Unread，不再新插，避免撞唯一键）
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

-- 会话表（type: 0 单聊 / 1 群聊 / 2 机器人通知，本阶段仅产生单聊）
CREATE TABLE `session` (
  `id` bigint NOT NULL COMMENT '雪花ID（会话ID）',
  `type` tinyint NOT NULL DEFAULT '0' COMMENT '0单聊 1群聊 2机器人',
  `name` varchar(64) NOT NULL DEFAULT '' COMMENT '会话名称（单聊为空，展示时取对端昵称）',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='会话表';

-- 用户-会话关联表（DR-4 决策：uk_session_user 唯一键防止并发通过申请产生重复关联行）
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
