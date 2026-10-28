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
