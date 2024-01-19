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


-- ----------------------------
-- Table structure for t_db_data_sync_task
-- ----------------------------
DROP TABLE IF EXISTS `t_db_data_sync_task`;
CREATE TABLE `t_db_data_sync_task`
(
    `id`                bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `creator_id`        bigint(20) NOT NULL COMMENT '创建人id',
    `creator`           varchar(100) NOT NULL COMMENT '创建人姓名',
    `create_time`       datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`       datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `modifier`          varchar(100) NOT NULL COMMENT '修改人姓名',
    `modifier_id`       bigint(20) NOT NULL COMMENT '修改人id',
    `task_name`         varchar(500) NOT NULL COMMENT '任务名',
    `task_cron`         varchar(50)  NOT NULL COMMENT '任务Cron表达式',
    `src_db_id`         bigint(20) NOT NULL COMMENT '源数据库ID',
    `src_db_name`       varchar(100)          DEFAULT NULL COMMENT '源数据库名',
    `src_tag_path`      varchar(200)          DEFAULT NULL COMMENT '源数据库tag路径',
    `target_db_id`      bigint(20) NOT NULL COMMENT '目标数据库ID',
    `target_db_name`    varchar(100)          DEFAULT NULL COMMENT '目标数据库名',
    `target_tag_path`   varchar(200)          DEFAULT NULL COMMENT '目标数据库tag路径',
    `target_table_name` varchar(100)          DEFAULT NULL COMMENT '目标数据库表名',
    `data_sql`          text         NOT NULL COMMENT '数据查询sql',
    `page_size`         int(11) NOT NULL COMMENT '数据同步分页大小',
    `upd_field`         varchar(100) NOT NULL DEFAULT 'id' COMMENT '更新字段，默认"id"',
    `upd_field_val`     varchar(100)          DEFAULT NULL COMMENT '当前更新值',
    `id_rule`           tinyint(2) NOT NULL DEFAULT '1' COMMENT 'id生成规则：1、MD5(时间戳+更新字段的值)。2、无(不自动生成id，选择无的时候需要指定主键ID字段是数据源哪个字段)',
    `pk_field`          varchar(100)          DEFAULT 'id' COMMENT '主键id字段名，默认"id"',
    `field_map`         text COMMENT '字段映射json',
    `is_deleted`        tinyint(8) DEFAULT '0',
    `delete_time`       datetime              DEFAULT NULL,
    `status`            tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态 1启用 2停用',
    `recent_state`      tinyint(1) NOT NULL DEFAULT '0' COMMENT '最近一次状态 0未执行 1成功 2失败',
    `task_key`          varchar(100)          DEFAULT NULL COMMENT '定时任务唯一uuid key',
    `running_state`     tinyint(1) DEFAULT '2' COMMENT '运行时状态 1运行中、2待运行、3已停止',
    PRIMARY KEY (`id`)
) COMMENT='数据同步';

-- ----------------------------
-- Table structure for t_db_data_sync_log
-- ----------------------------
DROP TABLE IF EXISTS `t_db_data_sync_log`;
CREATE TABLE `t_db_data_sync_log`
(
    `id`            bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `task_id`       bigint(20) NOT NULL COMMENT '同步任务表id',
    `create_time`   datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `data_sql_full` text     NOT NULL COMMENT '执行的完整sql',
    `res_num`       int(11) DEFAULT NULL COMMENT '收到数据条数',
    `err_text`      text COMMENT '错误日志',
    `status`        tinyint(4) NOT NULL DEFAULT '1' COMMENT '状态:1.成功  0.失败',
    `is_deleted`    tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除 1是 0 否',
    PRIMARY KEY (`id`),
    KEY             `t_db_data_sync_log_taskid_idx` (`task_id`) USING BTREE COMMENT 't_db_data_sync_log表(taskid)普通索引'
) COMMENT='数据同步日志';

INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`,
                              `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`,
                              `delete_time`)
VALUES (150, 36, 1, 1, '数据同步', 'sync', 1693040707,
        '{\"component\":\"ops/db/SyncTaskList\",\"icon\":\"Coin\",\"isKeepAlive\":true,\"routeName\":\"SyncTaskList\"}',
        12, 'liuzongyang', 12, 'liuzongyang', '2023-12-22 09:51:34', '2023-12-27 10:16:57', 'Jra0n7De/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`,
                              `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`,
                              `delete_time`)
VALUES (151, 150, 2, 1, '基本权限', 'db:sync', 1703641202, 'null', 12, 'liuzongyang', 12, 'liuzongyang',
        '2023-12-27 09:40:02', '2023-12-27 09:40:02', 'Jra0n7De/uAnHZxEV/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`,
                              `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`,
                              `delete_time`)
VALUES (152, 150, 2, 1, '编辑', 'db:sync:save', 1703641320, 'null', 12, 'liuzongyang', 12, 'liuzongyang',
        '2023-12-27 09:42:00', '2023-12-27 09:42:12', 'Jra0n7De/zvAMo2vk/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`,
                              `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`,
                              `delete_time`)
VALUES (153, 150, 2, 1, '删除', 'db:sync:del', 1703641342, 'null', 12, 'liuzongyang', 12, 'liuzongyang',
        '2023-12-27 09:42:22', '2023-12-27 09:42:22', 'Jra0n7De/pLOA2UYz/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`,
                              `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`,
                              `delete_time`)
VALUES (154, 150, 2, 1, '启停', 'db:sync:status', 1703641364, 'null', 12, 'liuzongyang', 12, 'liuzongyang',
        '2023-12-27 09:42:45', '2023-12-27 09:42:45', 'Jra0n7De/VBt68CDx/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`,
                              `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`,
                              `delete_time`)
VALUES (155, 150, 2, 1, '日志', 'db:sync:log', 1704266866, 'null', 12, 'liuzongyang', 12, 'liuzongyang',
        '2024-01-03 15:27:47', '2024-01-03 15:27:47', 'Jra0n7De/PigmSGVg/', 0, NULL);


DELETE
FROM `t_sys_config`
WHERE `key` = 'UseWatermark';

INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('系统全局样式设置', 'SysStyleConfig', '[{"model":"logoIcon","name":"logo图标","placeholder":"系统logo图标（base64编码, 建议svg格式，不超过10k）","required":false},{"model":"title","name":"菜单栏标题","placeholder":"系统菜单栏标题展示","required":false},{"model":"viceTitle","name":"登录页标题","placeholder":"登录页标题展示","required":false},{"model":"useWatermark","name":"是否启用水印","placeholder":"是否启用系统水印","options":"true,false","required":false},{"model":"watermarkContent","name":"水印补充信息","placeholder":"额外水印信息","required":false}]', '{"title":"mayfly-go","viceTitle":"mayfly-go","logoIcon":"","useWatermark":"true","watermarkContent":""}', '系统icon、标题、水印信息等配置', 'all', '2024-01-04 15:17:18', 1, 'admin', '2024-01-05 09:40:44', 1, 'admin', 0, NULL);
INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('数据库备份恢复', 'DbBackupRestore', '[{"model":"backupPath","name":"备份路径","placeholder":"备份文件存储路径"}]', '{"backupPath":"./db/backup"}', '', 'admin,', '2023-12-29 09:55:26', 1, 'admin', '2023-12-29 15:45:24', 1, 'admin', 0, NULL);
INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('Mysql可执行文件', 'MysqlBin', '[{"model":"path","name":"路径","placeholder":"可执行文件路径","required":true},{"model":"mysql","name":"mysql","placeholder":"mysql命令路径(空则为 路径/mysql)","required":false},{"model":"mysqldump","name":"mysqldump","placeholder":"mysqldump命令路径(空则为 路径/mysqldump)","required":false},{"model":"mysqlbinlog","name":"mysqlbinlog","placeholder":"mysqlbinlog命令路径(空则为 路径/mysqlbinlog)","required":false}]', '{"mysql":"","mysqldump":"","mysqlbinlog":"","path":"./db/mysql/bin"}', '', 'admin,', '2023-12-29 10:01:33', 1, 'admin', '2023-12-29 13:34:40', 1, 'admin', 0, NULL);
INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('MariaDB可执行文件', 'MariadbBin', '[{"model":"path","name":"路径","placeholder":"可执行文件路径","required":true},{"model":"mysql","name":"mysql","placeholder":"mysql命令路径(空则为 路径/mysql)","required":false},{"model":"mysqldump","name":"mysqldump","placeholder":"mysqldump命令路径(空则为 路径/mysqldump)","required":false},{"model":"mysqlbinlog","name":"mysqlbinlog","placeholder":"mysqlbinlog命令路径(空则为 路径/mysqlbinlog)","required":false}]', '{"mysql":"","mysqldump":"","mysqlbinlog":"","path":"./db/mariadb/bin"}', '', 'admin,', '2023-12-29 10:01:33', 1, 'admin', '2023-12-29 13:34:40', 1, 'admin', 0, NULL);


ALTER TABLE `t_db_instance`
    ADD COLUMN `sid` varchar(255) NULL COMMENT 'oracle数据库需要sid' AFTER `port`;