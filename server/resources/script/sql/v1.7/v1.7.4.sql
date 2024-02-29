begin;

-- 新增工单流程相关菜单
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1709208354, 1708911264, '6egfEVYr/fw0Hhvye/b4cNf3iq/', 2, 1, '删除流程', 'flow:procdef:del', 1709208354, 'null', 1, 'admin', 1, 'admin', '2024-02-29 20:05:54', '2024-02-29 20:05:54', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1709208339, 1708911264, '6egfEVYr/fw0Hhvye/r9ZMTHqC/', 2, 1, '保存流程', 'flow:procdef:save', 1709208339, 'null', 1, 'admin', 1, 'admin', '2024-02-29 20:05:40', '2024-02-29 20:05:40', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1709103180, 1708910975, '6egfEVYr/oNCIbynR/', 1, 1, '我的流程', 'procinsts', 1708911263, '{"component":"flow/ProcinstList","icon":"Tickets","isKeepAlive":true,"routeName":"ProcinstList"}', 1, 'admin', 1, 'admin', '2024-02-28 14:53:00', '2024-02-29 20:36:07', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1709045735, 1708910975, '6egfEVYr/3r3hHEub/', 1, 1, '我的任务', 'procinst-tasks', 1708911263, '{"component":"flow/ProcinstTaskList","icon":"Tickets","isKeepAlive":true,"routeName":"ProcinstTaskList"}', 1, 'admin', 1, 'admin', '2024-02-27 22:55:35', '2024-02-27 22:56:35', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1708911264, 1708910975, '6egfEVYr/fw0Hhvye/', 1, 1, '流程定义', 'procdefs', 1708911264, '{"component":"flow/ProcdefList","icon":"List","isKeepAlive":true,"routeName":"ProcdefList"}', 1, 'admin', 1, 'admin', '2024-02-26 09:34:24', '2024-02-27 22:54:32', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(1708910975, 0, '6egfEVYr/', 1, 1, '工单流程', '/flow', 60000000, '{"icon":"List","isKeepAlive":true,"routeName":"flow"}', 1, 'admin', 1, 'admin', '2024-02-26 09:29:36', '2024-02-26 15:37:52', 0, NULL);

-- 工单流程相关表
CREATE TABLE `t_flow_procdef` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `def_key` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '流程定义key',
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '流程名称',
  `status` tinyint DEFAULT NULL COMMENT '状态',
  `tasks` varchar(3000) COLLATE utf8mb4_bin NOT NULL COMMENT '审批节点任务信息',
  `remark` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `creator` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `creator_id` bigint NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `modifier_id` bigint NOT NULL,
  `is_deleted` tinyint DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='流程-流程定义';

CREATE TABLE `t_flow_procinst` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `procdef_id` bigint NOT NULL COMMENT '流程定义id',
  `procdef_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '流程定义名称',
  `task_key` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '当前任务key',
  `status` tinyint DEFAULT NULL COMMENT '状态',
  `biz_type` varchar(64) COLLATE utf8mb4_bin NOT NULL COMMENT '关联业务类型',
  `biz_key` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '关联业务key',
  `biz_status` tinyint DEFAULT NULL COMMENT '业务状态',
  `biz_handle_res` varchar(100) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '关联的业务处理结果',
  `remark` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `end_time` datetime DEFAULT NULL COMMENT '结束时间',
  `duration` bigint DEFAULT NULL COMMENT '流程持续时间（开始到结束）',
  `create_time` datetime NOT NULL COMMENT '流程发起时间',
  `creator` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '流程发起人',
  `creator_id` bigint NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `modifier_id` bigint NOT NULL,
  `is_deleted` tinyint DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_procdef_id` (`procdef_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='流程-流程实例(根据流程定义开启一个流程)';

CREATE TABLE `t_flow_procinst_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `procinst_id` bigint NOT NULL COMMENT '流程实例id',
  `task_key` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '任务key',
  `task_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '任务名称',
  `assignee` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '分配到该任务的用户',
  `status` tinyint DEFAULT NULL COMMENT '状态',
  `remark` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `end_time` datetime DEFAULT NULL COMMENT '结束时间',
  `duration` bigint DEFAULT NULL COMMENT '任务持续时间（开始到结束）',
  `create_time` datetime NOT NULL COMMENT '任务开始时间',
  `creator` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `creator_id` bigint NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `modifier_id` bigint NOT NULL,
  `is_deleted` tinyint DEFAULT '0',
  `delete_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_procinst_id` (`procinst_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='流程-流程实例任务';

-- 新增工单流程相关字段
ALTER TABLE t_db_sql_exec ADD status tinyint NULL COMMENT '执行状态';
ALTER TABLE t_db_sql_exec ADD flow_biz_key varchar(64) NULL COMMENT '工单流程定义key';
ALTER TABLE t_db_sql_exec ADD res varchar(1000) NULL COMMENT '执行结果';

ALTER TABLE t_db ADD flow_procdef_key varchar(64) NULL COMMENT '审批流-流程定义key（有值则说明关键操作需要进行审批执行）';

-- 历史执行记录调整为成功状态
UPDATE t_db_sql_exec SET status = 2

commit;