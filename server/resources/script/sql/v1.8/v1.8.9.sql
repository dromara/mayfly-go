ALTER TABLE `t_db_transfer_task`
    ADD COLUMN `task_name` varchar(100) NULL comment '任务名' after `delete_time`;