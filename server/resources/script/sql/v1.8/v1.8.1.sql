-- 关联数据库实例至标签
INSERT
	INTO
	t_tag_tree (code_path,
	code,
	type,
	NAME,
	create_time,
	creator_id,
	creator,
	update_time,
	modifier_id,
	modifier,
	is_deleted )
SELECT
	DISTINCT t1.newCodePath,
	t1.tagCode,
	100,
	t1.tagName,
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	0
FROM
	(
	SELECT
		ttt.code_path,
		ttt2.code_path as p_code_path,
		td.name,
		td.auth_cert_name,
		tdi.code tagCode,
		tdi.name tagName,
		CONCAT(ttt2.code_path, '2|', tdi.code, '/') newCodePath
	FROM
		t_tag_tree ttt
	JOIN t_db td ON
		ttt.code = td.code
		AND ttt.type = 2
		AND td.is_deleted = 0
	JOIN t_tag_tree ttt2 ON
		ttt.pid = ttt2.id
	JOIN t_db_instance tdi ON
		tdi.id = td.instance_id) AS t1;


-- 关联数据库实例的授权凭证信息至标签
INSERT
	INTO
	t_tag_tree (code_path,
	code,
	type,
	NAME,
	create_time,
	creator_id,
	creator,
	update_time,
	modifier_id,
	modifier,
	is_deleted )
SELECT
	DISTINCT t1.newCodePath,
	t1.tagCode,
	21,
	t1.tagName,
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	0
FROM
	(
	SELECT
		ttt.code_path,
		ttt2.code_path as p_code_path,
		td.name,
		td.auth_cert_name,
		trac.name tagCode,
		trac.username tagName,
		CONCAT(ttt2.code_path, '2|', tdi.code, '/21|', trac.name, '/') as newCodePath
	FROM
		t_tag_tree ttt
	JOIN t_db td ON
		ttt.code = td.code
		AND ttt.type = 2
		AND td.is_deleted = 0
	JOIN t_tag_tree ttt2 ON
		ttt.pid = ttt2.id
	JOIN t_db_instance tdi ON
		tdi.id = td.instance_id
	JOIN t_resource_auth_cert trac ON
		trac.name = td.auth_cert_name
		AND trac.is_deleted = 0) AS t1;

-- 关联数据库至标签
INSERT
	INTO
	t_tag_tree (code_path,
	code,
	type,
	NAME,
	create_time,
	creator_id,
	creator,
	update_time,
	modifier_id,
	modifier,
	is_deleted )
SELECT
	DISTINCT t1.newCodePath,
	t1.tagCode,
	22,
	t1.name,
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	DATE_FORMAT( NOW(), '%Y-%m-%d %H:%i:%s' ),
	1,
	'admin',
	0
FROM
	(
	SELECT
		ttt.code_path,
		ttt.code_path as p_code_path,
		td.name ,
		td.auth_cert_name,
		td.code tagCode,
		td.name tagName,
		CONCAT(ttt.code_path, '22|', td.code, '/') newCodePath
	FROM
		t_tag_tree ttt
	join t_db td on
		ttt.code = td.auth_cert_name
		and ttt.type = 21
		and td.is_deleted = 0) as t1;

UPDATE t_tag_tree SET is_deleted = 1, delete_time = NOW() WHERE `type` = 2;
UPDATE t_tag_tree SET `type` = 2 WHERE `type` = 100;

ALTER TABLE t_tag_tree DROP COLUMN pid;
ALTER TABLE t_tag_tree_team DROP COLUMN tag_path;