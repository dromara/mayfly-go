
-- DBMS配置变动
DELETE FROM `t_sys_config` WHERE `key` in ('DbQueryMaxCount', 'DbSaveQuerySQL');
INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('DBMS配置', 'DbmsConfig', '[{"model":"querySqlSave","name":"记录查询sql","placeholder":"是否记录查询类sql","options":"true,false"},{"model":"maxResultSet","name":"最大结果集","placeholder":"允许sql查询的最大结果集数。注: 0=不限制","options":""},{"model":"sqlExecTl","name":"sql执行时间限制","placeholder":"超过该时间（单位：秒），执行将被取消"}]', '{"querySqlSave":"false","maxResultSet":"0","sqlExecTl":"60"}', 'DBMS相关配置', 'admin,', '2024-03-06 13:30:51', 1, 'admin', '2024-03-06 14:07:16', 1, 'admin', 0, NULL);
ALTER TABLE `t_db_instance` CHANGE sid extra varchar(255) NULL COMMENT '连接需要的额外参数，如oracle数据库需要sid等';
ALTER TABLE `t_db_instance` MODIFY COLUMN extra varchar(255) NULL COMMENT '连接需要的额外参数，如oracle数据库需要sid等';

-- 数据迁移相关
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
ALTER TABLE t_sys_log ADD extra text NULL;
ALTER TABLE t_sys_log MODIFY COLUMN resp text NULL;

-- rdp相关
ALTER TABLE `t_machine` ADD COLUMN `protocol` tinyint(2) NULL COMMENT '协议  1、SSH  2、RDP' AFTER `name`;
update `t_machine` set `protocol` = 1 where `protocol`  is NULL;
delete from `t_sys_config` where `key` = 'MachineConfig';
INSERT INTO `t_sys_config` ( `name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('机器相关配置', 'MachineConfig', '[{"name":"终端回放存储路径","model":"terminalRecPath","placeholder":"终端回放存储路径"},{"name":"uploadMaxFileSize","model":"uploadMaxFileSize","placeholder":"允许上传的最大文件大小(1MB、2GB等)"},{"model":"termOpSaveDays","name":"终端记录保存时间","placeholder":"终端记录保存时间（单位天）"},{"model":"guacdHost","name":"guacd服务ip","placeholder":"guacd服务ip，默认 127.0.0.1","required":false},{"name":"guacd服务端口","model":"guacdPort","placeholder":"guacd服务端口，默认 4822","required":false},{"model":"guacdFilePath","name":"guacd服务文件存储位置","placeholder":"guacd服务文件存储位置，用于挂载RDP文件夹"},{"name":"guacd服务记录存储位置","model":"guacdRecPath","placeholder":"guacd服务记录存储位置，用于记录rdp操作记录"}]', '{"terminalRecPath":"./rec","uploadMaxFileSize":"1000MB","termOpSaveDays":"30","guacdHost":"","guacdPort":"","guacdFilePath":"./guacd/rdp-file","guacdRecPath":"./guacd/rdp-rec"}', '机器相关配置，如终端回放路径等', 'all', '2023-07-13 16:26:44', 1, 'admin', '2024-04-06 12:25:03', 1, 'admin', 0, NULL);

-- 授权凭证相关
CREATE TABLE `t_resource_auth_cert` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(100) DEFAULT NULL COMMENT '账号名称',
    `resource_code` varchar(36) DEFAULT NULL COMMENT '资源编码',
    `resource_type` tinyint NOT NULL COMMENT '资源类型',
    `type` tinyint DEFAULT NULL COMMENT '凭证类型',
    `username` varchar(100) DEFAULT NULL COMMENT '用户名',
    `ciphertext` varchar(5000) DEFAULT NULL COMMENT '密文内容',
    `ciphertext_type` tinyint NOT NULL COMMENT '密文类型（-1.公共授权凭证 1.密码 2.秘钥）',
    `extra` varchar(200) DEFAULT NULL COMMENT '账号需要的其他额外信息（如秘钥口令等）',
    `remark` varchar(50) DEFAULT NULL COMMENT '备注',
    `create_time` datetime NOT NULL,
    `creator_id` bigint NOT NULL,
    `creator` varchar(36) NOT NULL,
    `update_time` datetime NOT NULL,
    `modifier_id` bigint NOT NULL,
    `modifier` varchar(36) NOT NULL,
    `is_deleted` tinyint DEFAULT '0',
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_resource_code` (`resource_code`) USING BTREE,
    KEY `idx_name` (`name`) USING BTREE
) COMMENT='资源授权凭证表';

ALTER TABLE t_tag_tree ADD `type` tinyint NOT NULL DEFAULT '-1' COMMENT '类型： -1.普通标签； 其他值则为对应的资源类型' AFTER `code_path`;
ALTER TABLE t_db_instance ADD `code` varchar(36) NULL COMMENT '唯一编号' AFTER id;
ALTER TABLE t_db ADD auth_cert_name varchar(36) NULL COMMENT '授权凭证名' AFTER instance_id;
ALTER TABLE t_tag_tree MODIFY COLUMN code_path varchar(555) NOT NULL COMMENT '标识符路径';

BEGIN;
INSERT INTO t_tag_tree ( pid, CODE, code_path, type, NAME, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted )
SELECT
    tag_id,
    resource_code,
    CONCAT(tag_path ,resource_type , '|', resource_code, '/'),
    resource_type,
    resource_code,
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
    1,
    'admin',
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
    1,
    'admin',
    0
FROM
    t_tag_resource
WHERE
    is_deleted = 0;

-- 迁移machine表账号密码
INSERT INTO `t_resource_auth_cert` ( `name`, `resource_code`, `resource_type`, `username`, `ciphertext`, `ciphertext_type`, `type`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted` )
SELECT
    CONCAT( 'machine_', CODE, '_', username ) name,
    CODE resource_code,
    1 resource_type,
    username username,
    CASE
        WHEN auth_cert_id = - 1
            OR auth_cert_id IS NULL THEN
            `password` ELSE concat( 'auth_cert_', auth_cert_id )
        END ciphertext,
    CASE
        WHEN auth_cert_id = - 1
            OR auth_cert_id IS NULL THEN
            1 ELSE - 1
        END ciphertext_type,
    1 type,
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ) create_time,
    1 creator_id,
    'admin' creator,
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ) update_time,
    1 modifier_id,
    'admin' modifier,
    0 is_deleted
FROM
    t_machine
WHERE
    is_deleted = 0;

-- 迁移公共密钥
INSERT INTO `t_resource_auth_cert` ( `name`, `remark`, `resource_code`, `resource_type`, `username`, `ciphertext`, `extra`, `ciphertext_type`, `type`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted` )
SELECT
    concat( 'auth_cert_', id ) `name`,
    `name` remark,
    concat( 'auth_cert_code_', id ) resource_code,
    -2 resource_type,
    t.username username,
    `password` ciphertext,
    case when passphrase is not null and passphrase !='' then concat('{"passphrase":"', passphrase,'"}') else null end extra,
    auth_method ciphertext_type,
    2 type,
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ) create_time,
    1 creator_id,
    'admin' creator,
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ) update_time,
    1 modifier_id,
    'admin' modifier,
    0 is_deleted
FROM
    t_auth_cert a
        join (select `ciphertext`, `username` from `t_resource_auth_cert` GROUP BY `ciphertext`, `username` ) t on t.ciphertext = concat( 'auth_cert_', a.id )
;

-- 关联机器账号到tag_tree
INSERT INTO t_tag_tree ( pid, CODE, code_path, type, NAME, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted )
SELECT
    tt.id,
    rac.`name`,
    CONCAT(tt.code_path, '11|' ,rac.`name`, '/'),
    11,
    rac.`username`,
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
    1,
    'admin',
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
    1,
    'admin',
    0
FROM
    `t_tag_tree` tt
        JOIN `t_resource_auth_cert` rac ON tt.`code` = rac.`resource_code`
        AND tt.`type` = rac.`resource_type` AND rac.type = 1
WHERE
    tt.`is_deleted` = 0;

--  迁移数据库账号至授权凭证表
UPDATE t_db_instance SET `code` = CONCAT('db_code_', id);
INSERT INTO t_resource_auth_cert ( NAME, resource_code, resource_type, type, username, ciphertext, ciphertext_type, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted )
SELECT
    CONCAT( CODE, '_', username ),
    CODE,
    2,
    1,
    username,
    PASSWORD,
    1,
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
    1,
    'admin',
    DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
    1,
    'admin',
    0
FROM
    t_db_instance
WHERE
    is_deleted = 0;

UPDATE
    t_db d
SET
    d.auth_cert_name = (
        SELECT
            rac.name
        FROM
            t_resource_auth_cert rac
                join t_db_instance di on
                rac.resource_code = di.code
                    and rac.resource_type = 2
        WHERE
            di.id = d.instance_id);

UPDATE `t_sys_resource` SET pid=93, ui_path='Tag3fhad/exahgl32/', weight=19999999, meta='{"component":"ops/tag/AuthCertList","icon":"Ticket","isKeepAlive":true,"routeName":"AuthCertList"}' WHERE id=103;
UPDATE `t_sys_resource` SET ui_path='Tag3fhad/exahgl32/egxahg24/', weight=10000000 WHERE id=104;
UPDATE `t_sys_resource` SET ui_path='Tag3fhad/exahgl32/yglxahg2/', weight=20000000 WHERE id=105;
UPDATE `t_sys_resource` SET ui_path='Tag3fhad/exahgl32/Glxag234/', weight=30000000 WHERE id=106;
INSERT INTO `t_sys_resource` (`id`, `pid`, `ui_path`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `is_deleted`, `delete_time`) VALUES(1712717290, 0, 'tLb8TKLB/', 1, 1, '无页面权限', 'empty', 1712717290, '{"component":"empty","icon":"Menu","isHide":true,"isKeepAlive":true,"routeName":"empty"}', 1, 'admin', 1, 'admin', '2024-04-10 10:48:10', '2024-04-10 10:48:10', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `ui_path`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `is_deleted`, `delete_time`) VALUES(1712717337, 1712717290, 'tLb8TKLB/m2abQkA8/', 2, 1, '授权凭证密文查看', 'authcert:showciphertext', 1712717337, 'null', 1, 'admin', 1, 'admin', '2024-04-10 10:48:58', '2024-04-10 10:48:58', 0, NULL);
commit;

