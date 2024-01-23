ALTER TABLE `t_db_instance`
    MODIFY `port` int (8) NULL comment '数据库端口',
    MODIFY `username` varchar (255) NULL comment '数据库用户名';

INSERT INTO `mayfly-go`.`t_sys_resource` (`id`, `pid`, `ui_path`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `is_deleted`, `delete_time`)
    VALUES (161, 49, 'dbms23ax/xleaiec2/3NUXQFIO/', 2, 1, '数据库备份', 'db:backup', 1705973876, 'null', 1, 'admin', 1, 'admin', '2024-01-23 09:37:56', '2024-01-23 09:37:56', 0, NULL),
        (160, 49, 'dbms23ax/xleaiec2/ghErkTdb/', 2, 1, '数据库恢复', 'db:restore', 1705973909, 'null', 1, 'admin', 1, 'admin', '2024-01-23 09:38:29', '2024-01-23 09:38:29', 0, NULL);
