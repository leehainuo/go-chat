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
