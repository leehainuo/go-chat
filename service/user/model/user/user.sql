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
