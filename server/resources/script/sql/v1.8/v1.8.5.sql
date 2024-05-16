ALTER TABLE t_db ADD get_database_mode tinyint NULL COMMENT '库名获取方式（-1.实时获取、1.指定库名）';
UPDATE t_db SET get_database_mode = 1;


ALTER TABLE t_machine_cron_job_exec ADD machine_code varchar(36) NULL COMMENT '机器编号';
ALTER TABLE t_machine_cron_job_exec DROP COLUMN machine_id;
