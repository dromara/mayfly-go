ALTER TABLE `t_machine`
ADD COLUMN `extra` varchar(200) NULL comment '额外信息';

update t_sys_resource set meta='{"component":"system/role/RoleList","icon":"icon menu/role","isKeepAlive":true,"routeName":"RoleList"}' where id=11;
update t_sys_resource set meta='{"component":"system/account/AccountList","icon":"User","isKeepAlive":true,"routeName":"AccountList"}' where id=14;
update t_sys_resource set meta='{"component":"ops/db/SyncTaskList","icon":"Refresh","isKeepAlive":true,"routeName":"SyncTaskList"}' where id=150;
update t_sys_resource set '{"icon":"icon redis/redis","isKeepAlive":true,"routeName":"RDS"}' where id=60;
update t_sys_resource set '{"icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"Mongo"}' where id=79;