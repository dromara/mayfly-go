
DELETE
FROM `t_sys_config`
WHERE `key` in ('DbQueryMaxCount', 'DbSaveQuerySQL');

INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('DBMS配置', 'DbmsConfig', '[{"model":"querySqlSave","name":"记录查询sql","placeholder":"是否记录查询类sql","options":"true,false"},{"model":"maxResultSet","name":"最大结果集","placeholder":"允许sql查询的最大结果集数。注: 0=不限制","options":""},{"model":"sqlExecTl","name":"sql执行时间限制","placeholder":"超过该时间（单位：秒），执行将被取消"}]', '{"querySqlSave":"false","maxResultSet":"0","sqlExecTl":"60"}', 'DBMS相关配置', 'admin,', '2024-03-06 13:30:51', 1, 'admin', '2024-03-06 14:07:16', 1, 'admin', 0, NULL);

ALTER TABLE t_db_instance CHANGE sid extra varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL COMMENT '连接需要的额外参数，如oracle数据库需要sid等';
ALTER TABLE t_db_instance MODIFY COLUMN extra varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL COMMENT '连接需要的额外参数，如oracle数据库需要sid等';
