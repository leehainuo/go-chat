CREATE TABLE `user_balance` (
  `balance_id` bigint NOT NULL COMMENT '雪花ID',
  `user_id` bigint NOT NULL COMMENT '用户ID，1:1',
  `balance` bigint NOT NULL DEFAULT '0' COMMENT '总余额，单位：分',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`balance_id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户余额表';
