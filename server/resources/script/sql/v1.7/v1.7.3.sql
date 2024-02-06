ALTER TABLE `t_db_backup`
    ADD COLUMN `max_save_days` int(8) NOT NULL DEFAULT '0' COMMENT '最大保存天数' AFTER `interval`;

ALTER TABLE `t_db_binlog_history`
    ADD COLUMN `last_event_time` datetime NULL DEFAULT NULL COMMENT '最新事件时间' AFTER `first_event_time`;
