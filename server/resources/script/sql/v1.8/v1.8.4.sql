ALTER TABLE t_team ADD validity_start_date DATETIME NULL COMMENT '生效开始时间';
ALTER TABLE t_team ADD validity_end_date DATETIME NULL COMMENT '生效结束时间';

UPDATE t_team SET validity_start_date = NOW(), validity_end_date = '2034-01-01 00:00:00'