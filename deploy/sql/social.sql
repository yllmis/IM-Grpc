  CREATE DATABASE IF NOT EXISTS `social` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
  USE `social`;

  -- ----------------------------
  -- 1. 好友关系表
  -- ----------------------------
  CREATE TABLE `friends` (
    `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `user_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户ID',
    `friend_uid` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '好友ID',
    `remark` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '好友备注',
    `add_source` tinyint(4) DEFAULT NULL COMMENT '添加来源(如1:搜索, 2:群聊等)',
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '添加时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_friend` (`user_id`,`friend_uid`) -- 联合唯一索引，防止重复添加
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友关系表';

  -- ----------------------------
  -- 2. 好友申请表
  -- ----------------------------
  CREATE TABLE `friend_requests` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `user_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '申请人ID',
    `req_uid` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '被申请人ID',
    `req_msg` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '验证消息',
    `req_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
    `handle_result` tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '处理结果(1:未处理, 2:同意, 3:拒绝)',
    `handle_msg` varchar(255)  DEFAULT NULL COMMENT '处理附带消息',
    `handled_at` timestamp NULL DEFAULT NULL COMMENT '处理时间',
    PRIMARY KEY (`id`),
    KEY `idx_req_uid` (`req_uid`)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友申请表';

  -- ----------------------------
  -- 3. 群组表 (根据图1推断)
  -- 注意：图1显示群组的id是varchar(24)，可能是用的 MongoDB ObjectID 或者分布式短ID
  -- ----------------------------
  CREATE TABLE `groups` (
    `id` varchar(24) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群ID',
    `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群名称',
    `icon` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '群头像',
    `creator_uid` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群主ID',
    `status` tinyint(4) DEFAULT '1' COMMENT '状态(1:正常, 2:解散)',
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '建群时间',
    PRIMARY KEY (`id`)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群组表';

  -- ----------------------------
  -- 4. 群成员表 (根据图2推断)
  -- ----------------------------
  CREATE TABLE `group_members` (
    `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `group_id` varchar(24) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群ID',
    `user_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户ID',
    `role` tinyint(4) DEFAULT '0' COMMENT '角色(0:普通成员, 1:管理员, 2:群主)',
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '入群时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_group_user` (`group_id`,`user_id`) -- 联合唯一索引，防止同一个人重复加群
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群成员表';

  -- ----------------------------
  -- 5. 入群申请表 (根据图2推断)
  -- ----------------------------
  CREATE TABLE `group_requests` (
    `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `user_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '申请人ID',
    `group_id` varchar(24) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群ID',
    `req_msg` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '验证消息',
    `req_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
    `handle_result` tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '处理结果(1:未处理, 2:同意, 3:拒绝)',
    `handle_user_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '处理人ID(群主或管理员)',
    `handle_msg` varchar(255) DEFAULT NULL COMMENT '处理附带消息',
    `handled_at` timestamp NULL DEFAULT NULL COMMENT '处理时间',
    PRIMARY KEY (`id`),
    KEY `idx_group_id` (`group_id`)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='入群申请表';