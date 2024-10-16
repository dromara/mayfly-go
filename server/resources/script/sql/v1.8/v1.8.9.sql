ALTER TABLE `t_db_transfer_task`
    ADD COLUMN `task_name` varchar(100) NULL comment '任务名' after `delete_time`;

ALTER TABLE `t_flow_procdef`
    ADD COLUMN `condition` text NULL comment '触发审批的条件（计算结果返回1则需要启用该流程）';