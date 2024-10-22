
-- 数据同步新增字段
ALTER TABLE `t_db_data_sync_task`
    ADD COLUMN `upd_field_src` varchar(100) DEFAULT NULL COMMENT '更新值来源字段，默认同更新字段，如果查询结果指定了字段别名且与原更新字段不一致，则取这个字段值为当前更新值';

-- 新增数据库迁移到文件的菜单资源
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724376022, 1709194669, 2, 1, '文件-删除', 'db:transfer:files:del', 1724376022, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-23 09:20:23', '2024-08-23 14:50:21', 'SmLcpu6c/HIURtJJA/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724395850, 1709194669, 2, 1, '文件-下载', 'db:transfer:files:down', 1724395850, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-23 14:50:51', '2024-08-23 14:50:51', 'SmLcpu6c/FmqK4azt/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724398262, 1709194669, 2, 1, '文件', 'db:transfer:files', 1724376021, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-23 15:31:02', '2024-08-23 15:31:16', 'SmLcpu6c/btVtrbhk/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724998419, 1709194669, 2, 1, '文件-执行', 'db:transfer:files:run', 1724998419, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-30 14:13:39', '2024-08-30 14:13:39', 'SmLcpu6c/qINungml/', 0, NULL);

-- 新增数据库迁移相关的系统配置
DELETE FROM `t_sys_config` WHERE `key` = 'DbBackupRestore';
UPDATE `t_sys_config` SET params = '[{"name":"uploadMaxFileSize","model":"uploadMaxFileSize","placeholder":"允许上传的最大文件大小(1MB、2GB等)"},{"model":"termOpSaveDays","name":"终端记录保存时间","placeholder":"终端记录保存时间（单位天）"},{"model":"guacdHost","name":"guacd服务ip","placeholder":"guacd服务ip，默认 127.0.0.1","required":false},{"name":"guacd服务端口","model":"guacdPort","placeholder":"guacd服务端口，默认 4822","required":false},{"model":"guacdFilePath","name":"guacd服务文件存储位置","placeholder":"guacd服务文件存储位置，用于挂载RDP文件夹"}]' WHERE `key`='MachineConfig';
INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('文件配置', 'FileConfig', '[{"model":"basePath","name":"基础路径","placeholder":"默认为可执行文件对应目录下./file"}]', '{"basePath":"./file"}', '系统文件相关配置', 'admin,', '2024-10-20 22:30:01', 1, 'admin', '2024-10-21 13:51:17', 1, 'admin', 0, NULL);

-- 数据库迁移到文件
ALTER TABLE `t_db_transfer_task`
    ADD COLUMN `task_name` varchar(100) NULL comment '任务名',
    ADD COLUMN `cron_able` TINYINT(3) NOT NULL DEFAULT 0 comment '是否定时  1是 -1否',
    ADD COLUMN `cron` VARCHAR(20) NULL comment '定时任务cron表达式',
    ADD COLUMN `task_key` varchar(100) NULL comment '定时任务唯一uuid key',
    ADD COLUMN `mode` TINYINT(3) NOT NULL DEFAULT 1 comment '数据迁移方式，1、迁移到数据库  2、迁移到文件',
    ADD COLUMN `target_file_db_type` varchar(200) NULL comment '目标文件语言类型，类型枚举同target_db_type',
    ADD COLUMN `file_save_days` int NULL comment '文件保存天数',
    ADD COLUMN `status` tinyint(3) NOT NULL DEFAULT '1' comment '启用状态 1启用 -1禁用';

UPDATE `t_db_transfer_task` SET mode = 1 WHERE 1=1;
UPDATE `t_db_transfer_task` SET cron_able = -1 WHERE 1=1;
UPDATE `t_db_transfer_task` SET task_name = '未命名' WHERE task_name = '' or task_name is null;

CREATE TABLE `t_db_transfer_files` (
   `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
   `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
   `is_deleted` tinyint(3) NOT NULL DEFAULT 0 COMMENT '是否删除',
   `delete_time` datetime COMMENT '删除时间',
   `status` tinyint(3) NOT NULL DEFAULT 1 COMMENT '状态，1、执行中 2、执行失败 3、 执行成功',
   `task_id` bigint COMMENT '迁移任务ID',
   `log_id` bigint COMMENT '日志ID',
   `file_db_type` varchar(200) COMMENT 'sql文件数据库类型',
   `file_key` varchar(50) COMMENT '文件',
   PRIMARY KEY (id)
) COMMENT '数据库迁移文件管理';


ALTER TABLE `t_flow_procdef`
    ADD COLUMN `condition` text NULL comment '触发审批的条件（计算结果返回1则需要启用该流程）';

CREATE TABLE `t_sys_file`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `file_key` varchar(32)  NOT NULL COMMENT 'key',
  `filename` varchar(255)  NOT NULL COMMENT '文件名',
  `path` varchar(255)  NOT NULL COMMENT '文件路径',
  `size` int NULL DEFAULT NULL COMMENT '文件大小',
  `creator_id` bigint NULL DEFAULT NULL,
  `creator` varchar(32)  NULL DEFAULT NULL,
  `modifier_id` bigint NULL DEFAULT NULL,
  `modifier` varchar(255)  NULL DEFAULT NULL,
  `create_time` datetime NULL DEFAULT NULL,
  `update_time` datetime NULL DEFAULT NULL,
  `is_deleted` tinyint NOT NULL DEFAULT 0,
  `delete_time` datetime NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_file_key` (`file_key`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COMMENT = '系统文件表';