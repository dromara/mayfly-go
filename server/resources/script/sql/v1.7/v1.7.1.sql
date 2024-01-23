ALTER TABLE `t_db_instance`
    MODIFY `port` int (8) NULL comment '数据库端口',
    MODIFY `username` varchar (255) NULL comment '数据库用户名';
