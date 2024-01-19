begin;

DROP TABLE IF EXISTS `t_tag_resource`;
CREATE TABLE `t_tag_resource` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `tag_id` bigint NOT NULL,
  `tag_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '标签路径',
  `resource_code` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '资源编码',
  `resource_type` tinyint NOT NULL COMMENT '资源类型',
  `create_time` datetime NOT NULL,
  `creator_id` bigint NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint NOT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `is_deleted` tinyint DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tag_path` (`tag_path`(100)) USING BTREE,
  KEY `idx_resource_code` (`resource_code`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='标签资源关联表';

ALTER TABLE t_machine ADD COLUMN code varchar(32) NULL AFTER id;
CREATE INDEX idx_code USING BTREE ON t_machine (code);
UPDATE t_machine SET code = id;
INSERT
	INTO
	t_tag_resource (`tag_id`,
	`tag_path`,
	`resource_code`,
	`resource_type`,
	`create_time`,
	`creator_id`,
	`creator`,
	`update_time`,
	`modifier_id`,
	`modifier`,
	`is_deleted`,
	`delete_time`)
SELECT
	 `tag_id`,
	`tag_path`,
	`code`,
	"1",
	'2023-08-30 15:04:07',
	1,
	'admin',
	'2023-08-30 15:04:07',
	1,
	'admin',
	0,
	NULL
FROM
	t_machine
WHERE
	is_deleted = 0;


ALTER TABLE t_db ADD COLUMN code varchar(32) NULL AFTER id;
CREATE INDEX idx_code USING BTREE ON t_db (code);
UPDATE t_db SET code = id;
INSERT
	INTO
	t_tag_resource (`tag_id`,
	`tag_path`,
	`resource_code`,
	`resource_type`,
	`create_time`,
	`creator_id`,
	`creator`,
	`update_time`,
	`modifier_id`,
	`modifier`,
	`is_deleted`,
	`delete_time`)
SELECT
	 `tag_id`,
	`tag_path`,
	`code`,
	"2",
	'2023-08-30 15:04:07',
	1,
	'admin',
	'2023-08-30 15:04:07',
	1,
	'admin',
	0,
	NULL
FROM
	t_db
WHERE
	is_deleted = 0;


ALTER TABLE t_redis ADD COLUMN code varchar(32) NULL AFTER id;
CREATE INDEX idx_code USING BTREE ON t_redis (code);
UPDATE t_redis SET code = id;
INSERT
	INTO
	t_tag_resource (`tag_id`,
	`tag_path`,
	`resource_code`,
	`resource_type`,
	`create_time`,
	`creator_id`,
	`creator`,
	`update_time`,
	`modifier_id`,
	`modifier`,
	`is_deleted`,
	`delete_time`)
SELECT
	 `tag_id`,
	`tag_path`,
	`code`,
	"3",
	'2023-08-30 15:04:07',
	1,
	'admin',
	'2023-08-30 15:04:07',
	1,
	'admin',
	0,
	NULL
FROM
	t_redis
WHERE
	is_deleted = 0;


ALTER TABLE t_mongo ADD COLUMN code varchar(32) NULL AFTER id;
CREATE INDEX idx_code USING BTREE ON t_mongo (code);
UPDATE t_mongo SET code = id;
INSERT
	INTO
	t_tag_resource (`tag_id`,
	`tag_path`,
	`resource_code`,
	`resource_type`,
	`create_time`,
	`creator_id`,
	`creator`,
	`update_time`,
	`modifier_id`,
	`modifier`,
	`is_deleted`,
	`delete_time`)
SELECT
	 `tag_id`,
	`tag_path`,
	`code`,
	"4",
	'2023-08-30 15:04:07',
	1,
	'admin',
	'2023-08-30 15:04:07',
	1,
	'admin',
	0,
	NULL
FROM
	t_mongo
WHERE
	is_deleted = 0;

ALTER TABLE t_machine DROP COLUMN tag_id;
ALTER TABLE t_machine DROP COLUMN tag_path;

ALTER TABLE t_db DROP COLUMN tag_id;
ALTER TABLE t_db DROP COLUMN tag_path;

ALTER TABLE t_redis DROP COLUMN tag_id;
ALTER TABLE t_redis DROP COLUMN tag_path;

ALTER TABLE t_mongo DROP COLUMN tag_id;
ALTER TABLE t_mongo DROP COLUMN tag_path;

-- 机器终端操作记录表
DROP TABLE IF EXISTS `t_machine_term_op`;
CREATE TABLE `t_machine_term_op` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `machine_id` bigint NOT NULL COMMENT '机器id',
  `username` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '登录用户名',
  `record_file_path` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '终端回放文件路径',
  `creator_id` bigint unsigned DEFAULT NULL,
  `creator` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `end_time` datetime DEFAULT NULL,
  `is_deleted` tinyint DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='机器终端操作记录表';

commit;