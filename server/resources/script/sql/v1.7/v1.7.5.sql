
DELETE
FROM `t_sys_config`
WHERE `key` in ('DbQueryMaxCount', 'DbSaveQuerySQL');

INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('DBMS配置', 'DbmsConfig', '[{"model":"querySqlSave","name":"记录查询sql","placeholder":"是否记录查询类sql","options":"true,false"},{"model":"maxResultSet","name":"最大结果集","placeholder":"允许sql查询的最大结果集数。注: 0=不限制","options":""},{"model":"sqlExecTl","name":"sql执行时间限制","placeholder":"超过该时间（单位：秒），执行将被取消"}]', '{"querySqlSave":"false","maxResultSet":"0","sqlExecTl":"60"}', 'DBMS相关配置', 'admin,', '2024-03-06 13:30:51', 1, 'admin', '2024-03-06 14:07:16', 1, 'admin', 0, NULL);

ALTER TABLE t_db_instance CHANGE sid extra varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL COMMENT '连接需要的额外参数，如oracle数据库需要sid等';
ALTER TABLE t_db_instance MODIFY COLUMN extra varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL COMMENT '连接需要的额外参数，如oracle数据库需要sid等';


CREATE TABLE `t_db_transfer_task` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `creator_id` bigint(20) NOT NULL COMMENT '创建人id',
  `creator` varchar(100) NOT NULL COMMENT '创建人姓名',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `modifier_id` bigint(20) NOT NULL COMMENT '修改人id',
  `modifier` varchar(100) NOT NULL COMMENT '修改人姓名',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '是否删除',
  `delete_time` datetime DEFAULT NULL COMMENT '删除时间',
  `checked_keys` text NOT NULL COMMENT '选中需要迁移的表',
  `delete_table` tinyint(4) NOT NULL COMMENT '创建表前是否删除表  1是  -1否',
  `name_case` tinyint(4) NOT NULL COMMENT '表名、字段大小写转换  1无  2大写  3小写',
  `strategy` tinyint(4) NOT NULL COMMENT '迁移策略  1全量  2增量',
  `running_state` tinyint(1) DEFAULT '2' COMMENT '运行状态 1运行中  2待运行',
  `src_db_id` bigint(20) NOT NULL COMMENT '源库id',
  `src_db_name` varchar(200) NOT NULL COMMENT '源库名',
  `src_tag_path` varchar(200) NOT NULL COMMENT '源库tagPath',
  `src_db_type` varchar(200) NOT NULL COMMENT '源库类型',
  `src_inst_name` varchar(200) NOT NULL COMMENT '源库实例名',
  `target_db_id` bigint(20) NOT NULL COMMENT '目标库id',
  `target_db_name` varchar(200) NOT NULL COMMENT '目标库名',
  `target_tag_path` varchar(200) NOT NULL COMMENT '目标库类型',
  `target_db_type` varchar(200) NOT NULL COMMENT '目标库实例名',
  `target_inst_name` varchar(200) NOT NULL COMMENT '目标库tagPath',
  `log_id` bigint(20) NOT NULL COMMENT '日志id',
  PRIMARY KEY (`id`)
)  COMMENT='数据库迁移任务表';

INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1709194669, 36, 1, 1, '数据库迁移', 'transfer', 1709194669, '{"component":"ops/db/DbTransferList","icon":"Switch","isKeepAlive":true,"routeName":"DbTransferList"}', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-29 16:17:50', '2024-02-29 16:24:59', 'SmLcpu6c/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1709194694, 1709194669, 2, 1, '基本权限', 'db:transfer', 1709194694, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-29 16:18:14', '2024-02-29 16:18:14', 'SmLcpu6c/A9vAm4J8/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1709196697, 1709194669, 2, 1, '编辑', 'db:transfer:save', 1709196697, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-29 16:51:37', '2024-02-29 16:51:37', 'SmLcpu6c/5oJwPzNb/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1709196707, 1709194669, 2, 1, '删除', 'db:transfer:del', 1709196707, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-29 16:51:47', '2024-02-29 16:51:47', 'SmLcpu6c/L3ybnAEW/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1709196723, 1709194669, 2, 1, '启停', 'db:transfer:status', 1709196723, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-29 16:52:04', '2024-02-29 16:52:04', 'SmLcpu6c/hGiLN1VT/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1709196737, 1709194669, 2, 1, '日志', 'db:transfer:log', 1709196737, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-29 16:52:17', '2024-02-29 16:52:17', 'SmLcpu6c/CZhNIbWg/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1709196755, 1709194669, 2, 1, '运行', 'db:transfer:run', 1709196755, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-29 16:52:36', '2024-02-29 16:52:36', 'SmLcpu6c/b6yHt6V2/', 0, NULL);

ALTER TABLE t_sys_log ADD extra varchar(5000) NULL;
ALTER TABLE t_sys_log MODIFY COLUMN resp text NULL;




