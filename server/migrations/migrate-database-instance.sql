CREATE TABLE `t_db_instance` (
                              `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
                              `name` varchar(32) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '数据库实例名称',
                              `host` varchar(100) COLLATE utf8mb4_bin NOT NULL,
                              `port` int(8) NOT NULL,
                              `username` varchar(255) COLLATE utf8mb4_bin NOT NULL,
                              `password` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL,
                              `type` varchar(20) COLLATE utf8mb4_bin NOT NULL COMMENT '数据库实例类型(mysql...)',
                              `params` varchar(125) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '其他连接参数',
                              `network` varchar(20) COLLATE utf8mb4_bin DEFAULT NULL,
                              `ssh_tunnel_machine_id` bigint(20) DEFAULT NULL COMMENT 'ssh隧道的机器id',
                              `remark` varchar(125) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '备注，描述等',
                              `create_time` datetime DEFAULT NULL,
                              `creator_id` bigint(20) DEFAULT NULL,
                              `creator` varchar(32) COLLATE utf8mb4_bin DEFAULT NULL,
                              `update_time` datetime DEFAULT NULL,
                              `modifier_id` bigint(20) DEFAULT NULL,
                              `modifier` varchar(32) COLLATE utf8mb4_bin DEFAULT NULL,
                              `is_deleted` tinyint(8) NOT NULL DEFAULT '0',
                              `delete_time` datetime DEFAULT NULL,
                              PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='数据库实例信息表';

ALTER TABLE t_db
    ADD COLUMN instance_id bigint(20) UNSIGNED NULL AFTER tag_path;

BEGIN;

INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (135, 36, 'dbms23ax/X0f4BxT0/', 1, 1, '数据库实例管理', 'instances', 1693040706, '{\"component\":\"ops/db/InstanceList\",\"icon\":\"Coin\",\"isKeepAlive\":true,\"routeName\":\"InstanceList\"}', 1, 'admin', 1, 'admin', '2023-08-26 09:05:07', '2023-08-29 22:35:11', 0, NULL);

INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (136, 133, 'dbms23ax/X0f4BxT0/D23fUiBr/', 2, 1, '实例保存', 'instance:save', 1693041001, 'null', 1, 'admin', 1, 'admin', '2023-08-26 09:10:02', '2023-08-26 09:10:02', 0, NULL);

INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (137, 133, 'dbms23ax/X0f4BxT0/mJlBeTCs/', 2, 1, '基本权限', 'instance', 1693041055, 'null', 1, 'admin', 1, 'admin', '2023-08-26 09:10:55', '2023-08-26 09:10:55', 0, NULL);

INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (138, 133, 'dbms23ax/X0f4BxT0/Sgg8uPwz/', 2, 1, '实例删除', 'instance:del', 1693041084, 'null', 1, 'admin', 1, 'admin', '2023-08-26 09:11:24', '2023-08-26 09:11:24', 0, NULL);

INSERT INTO `t_sys_role_resource` (role_id,resource_id,creator_id,creator,create_time,is_deleted,delete_time) VALUES
                                                                                                                  (1,135,1,'admin','2023-08-30 20:17:00', 0, NULL),
                                                                                                                  (1,136,1,'admin','2023-08-30 20:17:00', 0, NULL),
                                                                                                                  (1,137,1,'admin','2023-08-30 20:17:00', 0, NULL),
                                                                                                                  (1,138,1,'admin','2023-08-30 20:17:00', 0, NULL);

INSERT INTO t_db_instance (`host`, `port`, `username`, `password`, `type`, `params`, `network`, `ssh_tunnel_machine_id`, `remark`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`)
SELECT DISTINCT `host`, `port`, `username`, `password`, `type`, `params`, `network`, `ssh_tunnel_machine_id`, '', '2023-08-30 15:04:07', 1, 'admin', '2023-08-30 15:04:07', 1, 'admin', 0, NULL
FROM t_db
WHERE is_deleted = 0;

UPDATE t_db_instance SET name = CONCAT('instance_', id)
WHERE name is NULL;

UPDATE t_db a, t_db_instance b SET a.instance_id = b.id
WHERE a.`host`=b.`host` and a.`port`=b.`port` and a.`username`=b.`username` and a.`password`=b.`password` and a.`type`=b.`type` and a.`params`=b.`params` and a.`network`=b.`network` and a.`ssh_tunnel_machine_id`=b.`ssh_tunnel_machine_id`;

COMMIT;

ALTER TABLE t_db
    MODIFY COLUMN instance_id bigint(20) UNSIGNED NOT NULL AFTER tag_path;

ALTER TABLE t_db
DROP COLUMN `host`,
DROP COLUMN `port`,
DROP COLUMN `username`,
DROP COLUMN `password`,
DROP COLUMN `type`,
DROP COLUMN `params`,
DROP COLUMN `network`,
DROP COLUMN `ssh_tunnel_machine_id`;