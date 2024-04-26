CREATE TABLE `t_tag_tree_relate` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `tag_id` bigint NOT NULL COMMENT '标签树id',
  `relate_id` bigint NOT NULL COMMENT '关联',
  `relate_type` tinyint NOT NULL COMMENT '关联类型',
  `create_time` datetime NOT NULL,
  `creator_id` bigint NOT NULL,
  `creator` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint NOT NULL,
  `modifier` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `is_deleted` tinyint NOT NULL DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tag_id` (`tag_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='与标签树有关联关系的表';


CREATE TABLE `t_machine_cmd_conf` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) COLLATE utf8_bin DEFAULT NULL COMMENT '名称',
  `cmds` varchar(500) COLLATE utf8_bin DEFAULT NULL COMMENT '命令配置',
  `status` tinyint(4) DEFAULT NULL COMMENT '状态',
  `stratege` varchar(100) COLLATE utf8_bin DEFAULT NULL COMMENT '策略',
  `remark` varchar(50) COLLATE utf8_bin DEFAULT NULL COMMENT '备注',
  `create_time` datetime NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(36) COLLATE utf8_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  `modifier` varchar(36) COLLATE utf8_bin NOT NULL,
  `is_deleted` tinyint(4) DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 COMMENT='机器命令配置';

INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1714032002, 1713875842, '12sSjal1/UnWIUhW0/0tJwC3Gf/', 2, 1, '命令配置-删除', 'cmdconf:del', 1714032002, 'null', 1, 'admin', 1, 'admin', '2024-04-25 16:00:02', '2024-04-25 16:00:02', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1714031981, 1713875842, '12sSjal1/UnWIUhW0/tEzIKecl/', 2, 1, '命令配置-保存', 'cmdconf:save', 1714031981, 'null', 1, 'admin', 1, 'admin', '2024-04-25 15:59:41', '2024-04-25 15:59:41', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1713875842, 2, '12sSjal1/UnWIUhW0/', 1, 1, '安全配置', 'security', 1713875842, '{"component":"ops/machine/security/SecurityConfList","icon":"Setting","isKeepAlive":true,"routeName":"SecurityConfList"}', 1, 'admin', 1, 'admin', '2024-04-23 20:37:22', '2024-04-23 20:37:22', 0, NULL);

INSERT
	INTO
	t_tag_tree_relate (tag_id,
	relate_id,
	relate_type,
	create_time,
	creator_id,
	creator,
	update_time,
	modifier_id,
	modifier,
	is_deleted )
SELECT
	tt.tag_id ,
	tt.team_id ,
	1,
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	0
FROM
	`t_tag_tree_team` tt
WHERE
	tt.`is_deleted` = 0;

DROP TABLE t_tag_tree_team;


CREATE TABLE `t_resource_op_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code_path` varchar(600) NOT NULL COMMENT '资源标签路径',
  `resource_code` varchar(32) NOT NULL COMMENT '资源编号',
  `resource_type` tinyint NOT NULL COMMENT '资源类型',
  `create_time` datetime NOT NULL,
  `creator_id` bigint NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `is_deleted` tinyint NOT NULL DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_resource_code` (`resource_code`) USING BTREE
) ENGINE=InnoDB COMMENT='资源操作记录';