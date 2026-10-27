-- gochat 数据库初始化脚本（phase-01 用户模块）
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
