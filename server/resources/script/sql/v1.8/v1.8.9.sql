
-- 数据同步新增字段
ALTER TABLE `t_db_data_sync_task`
    ADD COLUMN `upd_field_src` varchar(100) DEFAULT NULL COMMENT '更新值来源字段，默认同更新字段，如果查询结果指定了字段别名且与原更新字段不一致，则取这个字段值为当前更新值';

-- 新增数据库迁移到文件的菜单资源
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724376022, 1709194669, 2, 1, '文件-删除', 'db:transfer:files:del', 1724376022, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-23 09:20:23', '2024-08-23 14:50:21', 'SmLcpu6c/HIURtJJA/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724395850, 1709194669, 2, 1, '文件-下载', 'db:transfer:files:down', 1724395850, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-23 14:50:51', '2024-08-23 14:50:51', 'SmLcpu6c/FmqK4azt/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724398262, 1709194669, 2, 1, '文件', 'db:transfer:files', 1724376021, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-23 15:31:02', '2024-08-23 15:31:16', 'SmLcpu6c/btVtrbhk/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724817775, 1709194669, 2, 1, '文件-重命名', 'db:transfer:files:rename', 1724376021, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-28 12:02:56', '2024-08-28 12:03:01', 'SmLcpu6c/zu4fvnuA/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1724998419, 1709194669, 2, 1, '文件-执行', 'db:transfer:files:run', 1724998419, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-08-30 14:13:39', '2024-08-30 14:13:39', 'SmLcpu6c/qINungml/', 0, NULL);

-- 新增数据库迁移相关的系统配置
DELETE FROM `t_sys_config` WHERE `key` = 'DbBackupRestore';
INSERT INTO `t_sys_config` (`id`, `name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES(10, '数据库备份恢复', 'DbBackupRestore', '[{"model":"backupPath","name":"备份路径","placeholder":"备份文件存储路径"},{"model":"transferPath","name":"迁移路径","placeholder":"数据库迁移文件存储路径"}]', '{"backupPath":"./db/backup","transferPath":"./db/transfer"}', '数据库备份恢复', 'all', '2023-12-29 09:55:26', 1, 'admin', '2024-08-27 15:22:22', 12, 'liuzongyang', 0, NULL);

-- 数据库迁移到文件
ALTER TABLE `t_db_transfer_task`
    ADD COLUMN `task_name` varchar(100) NULL comment '任务名',
    ADD COLUMN `cron_able` TINYINT(3) NOT NULL DEFAULT 0 comment '是否定时  1是 -1否',
    ADD COLUMN `cron` VARCHAR(20) NULL comment '定时任务cron表达式',
    ADD COLUMN `task_key` varchar(100) NULL comment '定时任务唯一uuid key',
    ADD COLUMN `mode` TINYINT(3) NOT NULL DEFAULT 1 comment '数据迁移方式，1、迁移到数据库  2、迁移到文件',
    ADD COLUMN `target_file_db_type` varchar(200) NULL comment '目标文件语言类型，类型枚举同target_db_type',
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
   `file_name` varchar(200) COMMENT '显式文件名 默认: 年月日时分秒.zip',
   `file_uuid` varchar(50) COMMENT '文件真实uuid，拼接后可以下载',
   PRIMARY KEY (id)
) COMMENT '数据库迁移文件管理';


ALTER TABLE `t_flow_procdef`
    ADD COLUMN `condition` text NULL comment '触发审批的条件（计算结果返回1则需要启用该流程）';