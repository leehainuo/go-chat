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
