-- ----------------------------
-- Table structure for t_db_backup
-- ----------------------------
DROP TABLE IF EXISTS `t_db_backup`;
CREATE TABLE `t_db_backup` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(32) NOT NULL COMMENT '备份名称',
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `db_name` varchar(64) NOT NULL COMMENT '数据库名称',
    `repeated` tinyint(1) DEFAULT NULL COMMENT '是否重复执行',
    `interval` bigint(20) DEFAULT NULL COMMENT '备份周期',
    `start_time` datetime DEFAULT NULL COMMENT '首次备份时间',
    `enabled` tinyint(1) DEFAULT NULL COMMENT '是否启用',
    `last_status` tinyint(4) DEFAULT NULL COMMENT '上次备份状态',
    `last_result` varchar(256) DEFAULT NULL COMMENT '上次备份结果',
    `last_time` datetime DEFAULT NULL COMMENT '上次备份时间',
    `create_time` datetime DEFAULT NULL,
    `creator_id` bigint(20) unsigned DEFAULT NULL,
    `creator` varchar(32) DEFAULT NULL,
    `update_time` datetime DEFAULT NULL,
    `modifier_id` bigint(20) unsigned DEFAULT NULL,
    `modifier` varchar(32) DEFAULT NULL,
    `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_name` (`db_name`) USING BTREE,
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_backup_history
-- ----------------------------
DROP TABLE IF EXISTS `t_db_backup_history`;
CREATE TABLE `t_db_backup_history` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(64) NOT NULL COMMENT '历史备份名称',
    `db_backup_id` bigint(20) unsigned NOT NULL COMMENT '数据库备份ID',
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `db_name` varchar(64) NOT NULL COMMENT '数据库名称',
    `uuid` varchar(36) NOT NULL COMMENT '历史备份uuid',
    `binlog_file_name` varchar(32) DEFAULT NULL COMMENT 'BINLOG文件名',
    `binlog_sequence` bigint(20) DEFAULT NULL COMMENT 'BINLOG序列号',
    `binlog_position` bigint(20) DEFAULT NULL COMMENT 'BINLOG位置',
    `create_time` datetime DEFAULT NULL COMMENT '历史备份创建时间',
    `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_backup_id` (`db_backup_id`) USING BTREE,
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE,
    KEY `idx_db_name` (`db_name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_restore
-- ----------------------------
DROP TABLE IF EXISTS `t_db_restore`;
CREATE TABLE `t_db_restore` (
     `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
     `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
     `db_name` varchar(64) NOT NULL COMMENT '数据库名称',
     `repeated` tinyint(1) DEFAULT NULL COMMENT '是否重复执行',
     `interval` bigint(20) DEFAULT NULL COMMENT '恢复周期',
     `start_time` datetime DEFAULT NULL COMMENT '首次恢复时间',
     `enabled` tinyint(1) DEFAULT NULL COMMENT '是否启用',
     `last_status` tinyint(4) DEFAULT NULL COMMENT '上次恢复状态',
     `last_result` varchar(256) DEFAULT NULL COMMENT '上次恢复结果',
     `last_time` datetime DEFAULT NULL COMMENT '上次恢复时间',
     `point_in_time` datetime DEFAULT NULL COMMENT '恢复时间点',
     `db_backup_id` bigint(20) unsigned DEFAULT NULL COMMENT '备份ID',
     `db_backup_history_id` bigint(20) unsigned DEFAULT NULL COMMENT '历史备份ID',
     `db_backup_history_name` varchar(64) DEFAULT NULL COMMENT '历史备份名称',
     `create_time` datetime DEFAULT NULL,
     `creator_id` bigint(20) unsigned DEFAULT NULL,
     `creator` varchar(32) DEFAULT NULL,
     `update_time` datetime DEFAULT NULL,
     `modifier_id` bigint(20) unsigned DEFAULT NULL,
     `modifier` varchar(32) DEFAULT NULL,
     `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
     `delete_time` datetime DEFAULT NULL,
     PRIMARY KEY (`id`),
     KEY `idx_db_instane_id` (`db_instance_id`) USING BTREE,
     KEY `idx_db_name` (`db_name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=22 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_restore_history
-- ----------------------------
DROP TABLE IF EXISTS `t_db_restore_history`;
CREATE TABLE `t_db_restore_history` (
     `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
     `db_restore_id` bigint(20) unsigned NOT NULL COMMENT '恢复ID',
     `create_time` datetime DEFAULT NULL COMMENT '历史恢复创建时间',
     `is_deleted` tinyint(4) NOT NULL DEFAULT 0,
     `delete_time` datetime DEFAULT NULL,
     PRIMARY KEY (`id`),
     KEY `idx_db_restore_id` (`db_restore_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_binlog
-- ----------------------------
DROP TABLE IF EXISTS `t_db_binlog`;
CREATE TABLE `t_db_binlog` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `last_status` bigint(20) DEFAULT NULL COMMENT '上次下载状态',
    `last_result` varchar(256) DEFAULT NULL COMMENT '上次下载结果',
    `last_time` datetime DEFAULT NULL COMMENT '上次下载时间',
    `create_time` datetime DEFAULT NULL,
    `creator_id` bigint(20) unsigned DEFAULT NULL,
    `creator` varchar(32) DEFAULT NULL,
    `update_time` datetime DEFAULT NULL,
    `modifier_id` bigint(20) unsigned DEFAULT NULL,
    `modifier` varchar(32) DEFAULT NULL,
    `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_binlog_history
-- ----------------------------
DROP TABLE IF EXISTS `t_db_binlog_history`;
CREATE TABLE `t_db_binlog_history` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `file_name` varchar(32) DEFAULT NULL COMMENT 'BINLOG文件名称',
    `file_size` bigint(20) DEFAULT NULL COMMENT 'BINLOG文件大小',
    `sequence` bigint(20) DEFAULT NULL COMMENT 'BINLOG序列号',
    `first_event_time` datetime DEFAULT NULL COMMENT '首次事件时间',
    `create_time` datetime DEFAULT NULL,
    `is_deleted` tinyint(4) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
