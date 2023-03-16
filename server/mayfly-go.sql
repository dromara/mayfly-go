/*
 Navicat Premium Data Transfer

 Source Server Type    : MySQL
 Source Server Version : 50730
 Source Schema         : mayfly-go

 Target Server Type    : MySQL
 Target Server Version : 50730
 File Encoding         : 65001

 Date: 18/11/2021 14:33:55
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for t_db
-- ----------------------------
DROP TABLE IF EXISTS `t_db`;
CREATE TABLE `t_db` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '数据库实例名称',
  `host` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `port` int(8) NOT NULL,
  `username` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '数据库实例类型(mysql...)',
  `database` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '数据库,空格分割多个数据库',
  `params` varchar(125) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '其他连接参数',
  `network` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `ssh_tunnel_machine_id` bigint(20) DEFAULT NULL COMMENT 'ssh隧道的机器id',
  `remark` varchar(125) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '备注，描述等',
  `tag_id` bigint(20) DEFAULT NULL COMMENT '标签id',
  `tag_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '标签路径',
  `create_time` datetime DEFAULT NULL,
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_path` (`tag_path`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='数据库资源信息表';

-- ----------------------------
-- Records of t_db
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_db_sql
-- ----------------------------
DROP TABLE IF EXISTS `t_db_sql`;
CREATE TABLE `t_db_sql` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `db_id` bigint(20) NOT NULL COMMENT '数据库实例id',
  `db` varchar(125) COLLATE utf8mb4_bin NOT NULL COMMENT '数据库',
  `name` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'sql模板名',
  `sql` text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
  `type` tinyint(8) NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='数据库sql信息';

-- ----------------------------
-- Records of t_db_sql
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_db_sql_exec
-- ----------------------------
DROP TABLE IF EXISTS `t_db_sql_exec`;
CREATE TABLE `t_db_sql_exec` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `db_id` bigint(20) NOT NULL COMMENT '数据库id',
  `db` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '数据库',
  `table` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '表名',
  `type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'sql类型',
  `sql` varchar(5000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '执行sql',
  `old_value` varchar(5000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '操作前旧值',
  `remark` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '备注',
  `create_time` datetime NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='数据库sql执行记录表';

-- ----------------------------
-- Records of t_db_sql_exec
-- ----------------------------
BEGIN;
COMMIT;

DROP TABLE IF EXISTS `t_auth_cert`;
CREATE TABLE `t_auth_cert` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `auth_method` tinyint NOT NULL COMMENT '1.密码 2.秘钥',
  `password` varchar(4200) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '密码or私钥',
  `passphrase` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '私钥口令',
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `creator` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `creator_id` bigint NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier` varchar(12) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `modifier_id` bigint NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='授权凭证';

-- ----------------------------
-- Table structure for t_machine
-- ----------------------------
DROP TABLE IF EXISTS `t_machine`;
CREATE TABLE `t_machine` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `ip` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `port` int(12) NOT NULL,
  `username` varchar(12) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `auth_method` tinyint(2) DEFAULT NULL COMMENT '1.密码登录2.publickey登录',
  `password` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `auth_cert_id` bigint(20) DEFAULT NULL COMMENT '授权凭证id',
  `ssh_tunnel_machine_id` bigint(20) DEFAULT NULL COMMENT 'ssh隧道的机器id',
  `enable_recorder` tinyint(2) DEFAULT NULL COMMENT '是否启用终端回放记录',
  `status` tinyint(2) NOT NULL COMMENT '状态: 1:启用; -1:禁用',
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `tag_id` bigint(20) DEFAULT NULL COMMENT '标签id',
  `tag_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '标签路径',
  `need_monitor` tinyint(2) DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `creator` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `creator_id` bigint(32) DEFAULT NULL,
  `update_time` datetime NOT NULL,
  `modifier` varchar(12) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `modifier_id` bigint(32) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_path` (`tag_path`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='机器信息';

-- ----------------------------
-- Records of t_machine
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_machine_file
-- ----------------------------
DROP TABLE IF EXISTS `t_machine_file`;
CREATE TABLE `t_machine_file` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '机器文件配置（linux一切皆文件，故也可以表示目录）',
  `machine_id` bigint(20) NOT NULL,
  `name` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `path` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `type` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '1：目录；2：文件',
  `creator_id` bigint(20) unsigned DEFAULT NULL,
  `creator` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `modifier_id` bigint(20) unsigned DEFAULT NULL,
  `modifier` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=39 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='机器文件';

-- ----------------------------
-- Records of t_machine_file
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_machine_monitor
-- ----------------------------
DROP TABLE IF EXISTS `t_machine_monitor`;
CREATE TABLE `t_machine_monitor` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `machine_id` bigint(20) unsigned NOT NULL COMMENT '机器id',
  `cpu_rate` float(255,2) DEFAULT NULL,
  `mem_rate` float(255,2) DEFAULT NULL,
  `sys_load` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- ----------------------------
-- Records of t_machine_monitor
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_machine_script
-- ----------------------------
DROP TABLE IF EXISTS `t_machine_script`;
CREATE TABLE `t_machine_script` (
  `id` bigint(64) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '脚本名',
  `machine_id` bigint(64) NOT NULL COMMENT '机器id[0:公共]',
  `script` text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin COMMENT '脚本内容',
  `params` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '脚本入参',
  `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '脚本描述',
  `type` tinyint(8) DEFAULT NULL COMMENT '脚本类型[1: 有结果；2：无结果；3：实时交互]',
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=27 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='机器脚本';

-- ----------------------------
-- Records of t_machine_script
-- ----------------------------
BEGIN;
INSERT INTO `t_machine_script` VALUES (1, 'sys_info', 9999999, '# 获取系统cpu信息\nfunction get_cpu_info() {\n  Physical_CPUs=$(grep \"physical id\" /proc/cpuinfo | sort | uniq | wc -l)\n  Virt_CPUs=$(grep \"processor\" /proc/cpuinfo | wc -l)\n  CPU_Kernels=$(grep \"cores\" /proc/cpuinfo | uniq | awk -F \': \' \'{print $2}\')\n  CPU_Type=$(grep \"model name\" /proc/cpuinfo | awk -F \': \' \'{print $2}\' | sort | uniq)\n  CPU_Arch=$(uname -m)\n  echo -e \'\\n-------------------------- CPU信息 --------------------------\'\n  cat <<EOF | column -t\n物理CPU个数: $Physical_CPUs\n逻辑CPU个数: $Virt_CPUs\n每CPU核心数: $CPU_Kernels\nCPU型号: $CPU_Type\nCPU架构: $CPU_Arch\nEOF\n}\n\n# 获取系统信息\nfunction get_systatus_info() {\n  sys_os=$(uname -o)\n  sys_release=$(cat /etc/redhat-release)\n  sys_kernel=$(uname -r)\n  sys_hostname=$(hostname)\n  sys_selinux=$(getenforce)\n  sys_lang=$(echo $LANG)\n  sys_lastreboot=$(who -b | awk \'{print $3,$4}\')\n  echo -e \'-------------------------- 系统信息 --------------------------\'\n  cat <<EOF | column -t\n系统: ${sys_os}\n发行版本:   ${sys_release}\n系统内核:   ${sys_kernel}\n主机名:    ${sys_hostname}\nselinux状态:  ${sys_selinux}\n系统语言:   ${sys_lang}\n系统最后重启时间:   ${sys_lastreboot}\nEOF\n}\n\nget_systatus_info\n#echo -e \"\\n\"\nget_cpu_info', NULL, '获取系统信息', 1, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO `t_machine_script` VALUES (2, 'get_process_by_name', 9999999, '#! /bin/bash\n# Function: 根据输入的程序的名字过滤出所对应的PID，并显示出详细信息，如果有几个PID，则全部显示\nNAME={{.processName}}\nN=`ps -aux | grep $NAME | grep -v grep | wc -l`    ##统计进程总数\nif [ $N -le 0 ];then\n  echo \"无该进程！\"\nfi\ni=1\nwhile [ $N -gt 0 ]\ndo\n  echo \"进程PID: `ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $2}\'`\"\n  echo \"进程命令：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $11}\'`\"\n  echo \"进程所属用户: `ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $1}\'`\"\n  echo \"CPU占用率：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $3}\'`%\"\n  echo \"内存占用率：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $4}\'`%\"\n  echo \"进程开始运行的时刻：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $9}\'`\"\n  echo \"进程运行的时间：`  ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $11}\'`\"\n  echo \"进程状态：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $8}\'`\"\n  echo \"进程虚拟内存：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $5}\'`\"\n  echo \"进程共享内存：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $6}\'`\"\n  echo \"***************************************************************\"\n  let N-- i++\ndone', '[{\"name\": \"进程名\",\"model\": \"processName\", \"placeholder\": \"请输入进程名\"}]', '获取进程运行状态', 1, NULL, NULL, 1, 'admin', NULL, '2021-07-12 15:33:41');
INSERT INTO `t_machine_script` VALUES (3, 'sys_run_info', 9999999, '#!/bin/bash\n# 获取要监控的本地服务器IP地址\nIP=`ifconfig | grep inet | grep -vE \'inet6|127.0.0.1\' | awk \'{print $2}\'`\necho \"IP地址：\"$IP\n \n# 获取cpu总核数\ncpu_num=`grep -c \"model name\" /proc/cpuinfo`\necho \"cpu总核数：\"$cpu_num\n \n# 1、获取CPU利用率\n################################################\n#us 用户空间占用CPU百分比\n#sy 内核空间占用CPU百分比\n#ni 用户进程空间内改变过优先级的进程占用CPU百分比\n#id 空闲CPU百分比\n#wa 等待输入输出的CPU时间百分比\n#hi 硬件中断\n#si 软件中断\n#################################################\n# 获取用户空间占用CPU百分比\ncpu_user=`top -b -n 1 | grep Cpu | awk \'{print $2}\' | cut -f 1 -d \"%\"`\necho \"用户空间占用CPU百分比：\"$cpu_user\n \n# 获取内核空间占用CPU百分比\ncpu_system=`top -b -n 1 | grep Cpu | awk \'{print $4}\' | cut -f 1 -d \"%\"`\necho \"内核空间占用CPU百分比：\"$cpu_system\n \n# 获取空闲CPU百分比\ncpu_idle=`top -b -n 1 | grep Cpu | awk \'{print $8}\' | cut -f 1 -d \"%\"`\necho \"空闲CPU百分比：\"$cpu_idle\n \n# 获取等待输入输出占CPU百分比\ncpu_iowait=`top -b -n 1 | grep Cpu | awk \'{print $10}\' | cut -f 1 -d \"%\"`\necho \"等待输入输出占CPU百分比：\"$cpu_iowait\n \n#2、获取CPU上下文切换和中断次数\n# 获取CPU中断次数\ncpu_interrupt=`vmstat -n 1 1 | sed -n 3p | awk \'{print $11}\'`\necho \"CPU中断次数：\"$cpu_interrupt\n \n# 获取CPU上下文切换次数\ncpu_context_switch=`vmstat -n 1 1 | sed -n 3p | awk \'{print $12}\'`\necho \"CPU上下文切换次数：\"$cpu_context_switch\n \n#3、获取CPU负载信息\n# 获取CPU15分钟前到现在的负载平均值\ncpu_load_15min=`uptime | awk \'{print $11}\' | cut -f 1 -d \',\'`\necho \"CPU 15分钟前到现在的负载平均值：\"$cpu_load_15min\n \n# 获取CPU5分钟前到现在的负载平均值\ncpu_load_5min=`uptime | awk \'{print $10}\' | cut -f 1 -d \',\'`\necho \"CPU 5分钟前到现在的负载平均值：\"$cpu_load_5min\n \n# 获取CPU1分钟前到现在的负载平均值\ncpu_load_1min=`uptime | awk \'{print $9}\' | cut -f 1 -d \',\'`\necho \"CPU 1分钟前到现在的负载平均值：\"$cpu_load_1min\n \n# 获取任务队列(就绪状态等待的进程数)\ncpu_task_length=`vmstat -n 1 1 | sed -n 3p | awk \'{print $1}\'`\necho \"CPU任务队列长度：\"$cpu_task_length\n \n#4、获取内存信息\n# 获取物理内存总量\nmem_total=`free -h | grep Mem | awk \'{print $2}\'`\necho \"物理内存总量：\"$mem_total\n \n# 获取操作系统已使用内存总量\nmem_sys_used=`free -h | grep Mem | awk \'{print $3}\'`\necho \"已使用内存总量(操作系统)：\"$mem_sys_used\n \n# 获取操作系统未使用内存总量\nmem_sys_free=`free -h | grep Mem | awk \'{print $4}\'`\necho \"剩余内存总量(操作系统)：\"$mem_sys_free\n \n# 获取应用程序已使用的内存总量\nmem_user_used=`free | sed -n 3p | awk \'{print $3}\'`\necho \"已使用内存总量(应用程序)：\"$mem_user_used\n \n# 获取应用程序未使用内存总量\nmem_user_free=`free | sed -n 3p | awk \'{print $4}\'`\necho \"剩余内存总量(应用程序)：\"$mem_user_free\n \n# 获取交换分区总大小\nmem_swap_total=`free | grep Swap | awk \'{print $2}\'`\necho \"交换分区总大小：\"$mem_swap_total\n \n# 获取已使用交换分区大小\nmem_swap_used=`free | grep Swap | awk \'{print $3}\'`\necho \"已使用交换分区大小：\"$mem_swap_used\n \n# 获取剩余交换分区大小\nmem_swap_free=`free | grep Swap | awk \'{print $4}\'`\necho \"剩余交换分区大小：\"$mem_swap_free', NULL, '获取cpu、内存等系统运行状态', 1, NULL, NULL, NULL, NULL, NULL, '2021-04-25 15:07:16');
INSERT INTO `t_machine_script` VALUES (4, 'top', 9999999, 'top', NULL, '实时获取系统运行状态', 3, NULL, NULL, 1, 'admin', NULL, '2021-05-24 15:58:20');
INSERT INTO `t_machine_script` VALUES (18, 'disk-mem', 9999999, 'df -h', '', '磁盘空间查看', 1, 1, 'admin', 1, 'admin', '2021-07-16 10:49:53', '2021-07-16 10:49:53');
COMMIT;

-- ----------------------------
-- Table structure for t_mongo
-- ----------------------------
DROP TABLE IF EXISTS `t_mongo`;
CREATE TABLE `t_mongo` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '名称',
  `uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '连接uri',
  `ssh_tunnel_machine_id` bigint(20) DEFAULT NULL COMMENT 'ssh隧道的机器id',
  `tag_id` bigint(20) DEFAULT NULL COMMENT '标签id',
  `tag_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '标签路径',
  `create_time` datetime NOT NULL,
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- ----------------------------
-- Records of t_mongo
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_redis
-- ----------------------------
DROP TABLE IF EXISTS `t_redis`;
CREATE TABLE `t_redis` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '名称',
  `host` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `password` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `db` varchar(64) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '库号: 多个库用,分割',
  `mode` varchar(32) COLLATE utf8mb4_bin DEFAULT NULL,
  `ssh_tunnel_machine_id` bigint(20) DEFAULT NULL COMMENT 'ssh隧道的机器id',
  `remark` varchar(125) COLLATE utf8mb4_bin DEFAULT NULL,
  `tag_id` bigint(20) DEFAULT NULL COMMENT '标签id',
  `tag_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '标签路径',
  `creator` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `creator_id` bigint(32) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `modifier` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tag_path` (`tag_path`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='redis信息';

-- ----------------------------
-- Records of t_redis
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_sys_account
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_account`;
CREATE TABLE `t_sys_account` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `username` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `password` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `status` tinyint(4) DEFAULT NULL,
  `last_login_time` datetime DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` varchar(24) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `creator_id` bigint(255) NOT NULL,
  `creator` varchar(12) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(255) NOT NULL,
  `modifier` varchar(12) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='账号信息表';

-- ----------------------------
-- Records of t_sys_account
-- ----------------------------
BEGIN;
INSERT INTO `t_sys_account` VALUES (1, "管理员", 'admin', '$2a$10$w3Wky2U.tinvR7c/s0aKPuwZsIu6pM1/DMJalwBDMbE6niHIxVrrm', 1, '2022-10-26 20:03:48', '::1', '2020-01-01 19:00:00', 1, 'admin', '2020-01-01 19:00:00', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_sys_account_role
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_account_role`;
CREATE TABLE `t_sys_account_role` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT 'Id',
  `account_id` bigint(20) NOT NULL COMMENT '账号id',
  `role_id` bigint(20) NOT NULL COMMENT '角色id',
  `creator` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `creator_id` bigint(20) unsigned DEFAULT NULL,
  `create_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=43 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='账号角色关联表';

-- ----------------------------
-- Records of t_sys_account_role
-- ----------------------------
BEGIN;
INSERT INTO `t_sys_account_role` VALUES (25, 1, 1, 'admin', 1, '2021-05-28 16:21:45');
COMMIT;

-- ----------------------------
-- Table structure for t_sys_config
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_config`;
CREATE TABLE `t_sys_config` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '配置名',
  `key` varchar(120) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '配置key',
  `params` varchar(500) COLLATE utf8mb4_bin DEFAULT NULL,
  `value` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '配置value',
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '备注',
  `create_time` datetime NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- ----------------------------
-- Records of t_sys_config
-- ----------------------------
BEGIN;
INSERT INTO `t_sys_config` VALUES (1, '是否启用登录验证码', 'UseLoginCaptcha', NULL, '1', '1: 启用、0: 不启用', '2022-08-25 22:27:17', 1, 'admin', '2022-08-26 10:26:56', 1, 'admin');
INSERT INTO `t_sys_config` VALUES (2, '是否启用水印', 'UseWartermark', NULL, '1', '1: 启用、0: 不启用', '2022-08-25 23:36:35', 1, 'admin', '2022-08-26 10:02:52', 1, 'admin');
INSERT INTO `t_sys_config` VALUES (3, '数据库查询最大结果集', 'DbQueryMaxCount', '[]', '200', '允许sql查询的最大结果集数。注: 0=不限制', '2023-02-11 14:29:03', 1, 'admin', '2023-02-11 14:40:56', 1, 'admin');
INSERT INTO `t_sys_config` VALUES (4, '数据库是否记录查询SQL', 'DbSaveQuerySQL', '[]', '0', '1: 记录、0:不记录', '2023-02-11 16:07:14', 1, 'admin', '2023-02-11 16:44:17', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_sys_log
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_log`;
CREATE TABLE `t_sys_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `type` tinyint(4) NOT NULL COMMENT '类型',
  `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '描述',
  `req_param` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '请求信息',
  `resp` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '响应信息',
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '调用者',
  `creator_id` bigint(20) NOT NULL COMMENT '调用者id',
  `create_time` datetime NOT NULL COMMENT '操作时间',
  PRIMARY KEY (`id`),
  KEY `idx_creator_id` (`creator_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=58 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='系统操作日志';

-- ----------------------------
-- Records of t_sys_log
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_sys_msg
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_msg`;
CREATE TABLE `t_sys_msg` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `type` int(255) DEFAULT NULL,
  `msg` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `recipient_id` bigint(20) DEFAULT NULL COMMENT '接收人id，-1为所有接收',
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=91 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='系统消息表';

-- ----------------------------
-- Records of t_sys_msg
-- ----------------------------
BEGIN;
COMMIT;

-- ----------------------------
-- Table structure for t_sys_resource
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_resource`;
CREATE TABLE `t_sys_resource` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `pid` int(11) NOT NULL COMMENT '父节点id',
  `type` tinyint(255) NOT NULL COMMENT '1：菜单路由；2：资源（按钮等）',
  `status` int(255) NOT NULL COMMENT '状态；1:可用，-1:禁用',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '名称',
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '菜单路由为path，其他为唯一标识',
  `weight` int(11) DEFAULT NULL COMMENT '权重顺序',
  `meta` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '元数据',
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  `modifier` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=103 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='资源表';

-- ----------------------------
-- Records of t_sys_resource
-- ----------------------------
BEGIN;
INSERT INTO `t_sys_resource` (id,pid,`type`,status,name,code,weight,meta,creator_id,creator,modifier_id,modifier,create_time,update_time) VALUES
	 (1,0,1,1,'首页','/home',1,'{"component":"home/Home","icon":"HomeFilled","isAffix":true,"isKeepAlive":true,"routeName":"Home"}',1,'admin',1,'admin','2021-05-25 16:44:41','2023-03-14 14:27:07'),
	 (2,0,1,1,'机器管理','/machine',4,'{"icon":"Monitor","isKeepAlive":true,"redirect":"machine/list","routeName":"Machine"}',1,'admin',1,'admin','2021-05-25 16:48:16','2022-10-06 14:58:49'),
	 (3,2,1,1,'机器列表','machines',2,'{"component":"ops/machine/MachineList","icon":"Monitor","isKeepAlive":true,"routeName":"MachineList"}',2,'admin',1,'admin','2021-05-25 16:50:04','2023-03-15 17:14:44'),
	 (4,0,1,1,'系统管理','/sys',8,'{"icon":"Setting","isKeepAlive":true,"redirect":"/sys/resources","routeName":"sys"}',1,'admin',1,'admin','2021-05-26 15:20:20','2022-10-06 14:59:53'),
	 (5,4,1,1,'资源管理','resources',3,'{"component":"system/resource/ResourceList","icon":"Menu","isKeepAlive":true,"routeName":"ResourceList"}',1,'admin',1,'admin','2021-05-26 15:23:07','2023-03-14 15:44:34'),
	 (11,4,1,1,'角色管理','roles',2,'{"component":"system/role/RoleList","icon":"Menu","isKeepAlive":true,"routeName":"RoleList"}',1,'admin',1,'admin','2021-05-27 11:15:35','2023-03-14 15:44:22'),
	 (12,3,2,1,'机器终端按钮','machine:terminal',4,'',1,'admin',1,'admin','2021-05-28 14:06:02','2021-05-31 17:47:59'),
	 (14,4,1,1,'账号管理','accounts',1,'{"component":"system/account/AccountList","icon":"Menu","isKeepAlive":true,"routeName":"AccountList"}',1,'admin',1,'admin','2021-05-28 14:56:25','2023-03-14 15:44:10'),
	 (15,3,2,1,'文件管理按钮','machine:file',5,NULL,1,'admin',1,'admin','2021-05-31 17:44:37','2021-05-31 17:48:07'),
	 (16,3,2,1,'机器添加按钮','machine:add',1,NULL,1,'admin',1,'admin','2021-05-31 17:46:11','2021-05-31 19:34:15'),
	 (17,3,2,1,'机器编辑按钮','machine:update',2,NULL,1,'admin',1,'admin','2021-05-31 17:46:23','2021-05-31 19:34:18'),
	 (18,3,2,1,'机器删除按钮','machine:del',3,NULL,1,'admin',1,'admin','2021-05-31 17:46:36','2021-05-31 19:34:17'),
	 (19,14,2,1,'角色分配按钮','account:saveRoles',1,NULL,1,'admin',1,'admin','2021-05-31 17:50:51','2021-05-31 19:19:30'),
	 (20,11,2,1,'分配菜单&权限按钮','role:saveResources',1,NULL,1,'admin',1,'admin','2021-05-31 17:51:41','2021-05-31 19:33:37'),
	 (21,14,2,1,'账号删除按钮','account:del',2,'null',1,'admin',1,'admin','2021-05-31 18:02:01','2021-06-10 17:12:17'),
	 (22,11,2,1,'角色删除按钮','role:del',2,NULL,1,'admin',1,'admin','2021-05-31 18:02:29','2021-05-31 19:33:38'),
	 (23,11,2,1,'角色新增按钮','role:add',3,NULL,1,'admin',1,'admin','2021-05-31 18:02:44','2021-05-31 19:33:39'),
	 (24,11,2,1,'角色编辑按钮','role:update',4,NULL,1,'admin',1,'admin','2021-05-31 18:02:57','2021-05-31 19:33:40'),
	 (25,5,2,1,'资源新增按钮','resource:add',1,NULL,1,'admin',1,'admin','2021-05-31 18:03:33','2021-05-31 19:31:47'),
	 (26,5,2,1,'资源删除按钮','resource:delete',2,NULL,1,'admin',1,'admin','2021-05-31 18:03:47','2021-05-31 19:29:40'),
	 (27,5,2,1,'资源编辑按钮','resource:update',3,NULL,1,'admin',1,'admin','2021-05-31 18:04:03','2021-05-31 19:29:40'),
	 (28,5,2,1,'资源禁用启用按钮','resource:changeStatus',4,NULL,1,'admin',1,'admin','2021-05-31 18:04:33','2021-05-31 18:04:33'),
	 (29,14,2,1,'账号添加按钮','account:add',3,NULL,1,'admin',1,'admin','2021-05-31 19:23:42','2021-05-31 19:23:42'),
	 (30,14,2,1,'账号编辑修改按钮','account:update',4,NULL,1,'admin',1,'admin','2021-05-31 19:23:58','2021-05-31 19:23:58'),
	 (31,14,2,1,'账号管理基本权限','account',0,'null',1,'admin',1,'admin','2021-05-31 21:25:06','2021-06-22 11:20:34'),
	 (32,5,2,1,'资源管理基本权限','resource',0,NULL,1,'admin',1,'admin','2021-05-31 21:25:25','2021-05-31 21:25:25'),
	 (33,11,2,1,'角色管理基本权限','role',0,NULL,1,'admin',1,'admin','2021-05-31 21:25:40','2021-05-31 21:25:40'),
	 (34,14,2,1,'账号启用禁用按钮','account:changeStatus',5,NULL,1,'admin',1,'admin','2021-05-31 21:29:48','2021-05-31 21:29:48'),
	 (36,0,1,1,'DBMS','/dbms',5,'{"icon":"Coin","isKeepAlive":true,"routeName":"DBMS"}',1,'admin',1,'admin','2021-06-01 14:01:33','2023-03-15 17:31:08'),
	 (37,3,2,1,'添加文件配置','machine:addFile',6,'null',1,'admin',1,'admin','2021-06-01 19:54:23','2021-06-01 19:54:23'),
	 (38,36,1,1,'数据操作','sql-exec',1,'{"component":"ops/db/SqlExec","icon":"Coin","isKeepAlive":true,"routeName":"SqlExec"}',1,'admin',1,'admin','2021-06-03 09:09:29','2023-03-15 17:31:21'),
	 (39,0,1,1,'个人中心','/personal',2,'{"component":"personal/index","icon":"UserFilled","isHide":true,"isKeepAlive":true,"routeName":"Personal"}',1,'admin',1,'admin','2021-06-03 14:25:35','2023-03-14 14:28:36'),
	 (40,3,2,1,'文件管理-新增按钮','machine:file:add',7,'null',1,'admin',1,'admin','2021-06-08 11:06:26','2021-06-08 11:12:28'),
	 (41,3,2,1,'文件管理-删除按钮','machine:file:del',8,'null',1,'admin',1,'admin','2021-06-08 11:06:49','2021-06-08 11:06:49'),
	 (42,3,2,1,'文件管理-写入or下载文件权限','machine:file:write',9,'null',1,'admin',1,'admin','2021-06-08 11:07:27','2021-06-08 11:07:27'),
	 (43,3,2,1,'文件管理-文件上传按钮','machine:file:upload',10,'null',1,'admin',1,'admin','2021-06-08 11:07:42','2021-06-08 11:07:42'),
	 (44,3,2,1,'文件管理-删除文件按钮','machine:file:rm',11,'null',1,'admin',1,'admin','2021-06-08 11:08:12','2021-06-08 11:08:12'),
	 (45,3,2,1,'脚本管理-保存脚本按钮','machine:script:save',12,'null',1,'admin',1,'admin','2021-06-08 11:09:01','2021-06-08 11:09:01'),
	 (46,3,2,1,'脚本管理-删除按钮','machine:script:del',13,'null',1,'admin',1,'admin','2021-06-08 11:09:27','2021-06-08 11:09:27'),
	 (47,3,2,1,'脚本管理-执行按钮','machine:script:run',14,'null',1,'admin',1,'admin','2021-06-08 11:09:50','2021-06-08 11:09:50'),
	 (49,36,1,1,'数据库管理','dbs',2,'{"component":"ops/db/DbList","icon":"Coin","isKeepAlive":true,"routeName":"DbList"}',1,'admin',1,'admin','2021-07-07 15:13:55','2023-03-15 17:31:28'),
	 (54,49,2,1,'数据库保存','db:save',1,'null',1,'admin',1,'admin','2021-07-08 17:30:36','2021-07-08 17:31:05'),
	 (55,49,2,1,'数据库删除','db:del',2,'null',1,'admin',1,'admin','2021-07-08 17:30:48','2021-07-08 17:30:48'),
	 (57,3,2,1,'基本权限','machine',0,'null',1,'admin',1,'admin','2021-07-09 10:48:02','2021-07-09 10:48:02'),
	 (58,49,2,1,'基本权限','db',0,'null',1,'admin',1,'admin','2021-07-09 10:48:22','2021-07-09 10:48:22'),
	 (59,38,2,1,'基本权限','db:exec',1,'null',1,'admin',1,'admin','2021-07-09 10:50:13','2021-07-09 10:50:13'),
	 (60,0,1,1,'Redis','/redis',6,'{"icon":"iconfont icon-redis","isKeepAlive":true,"routeName":"RDS"}',1,'admin',1,'admin','2021-07-19 20:15:41','2023-03-15 16:44:59'),
	 (61,60,1,1,'数据操作','data-operation',1,'{"component":"ops/redis/DataOperation","icon":"iconfont icon-redis","isKeepAlive":true,"routeName":"DataOperation"}',1,'admin',1,'admin','2021-07-19 20:17:29','2023-03-15 16:37:50'),
	 (62,61,2,1,'基本权限','redis:data',1,'null',1,'admin',1,'admin','2021-07-19 20:18:54','2021-07-19 20:18:54'),
	 (63,60,1,1,'redis管理','manage',2,'{"component":"ops/redis/RedisList","icon":"iconfont icon-redis","isKeepAlive":true,"routeName":"RedisList"}',1,'admin',1,'admin','2021-07-20 10:48:04','2023-03-15 16:38:00'),
	 (64,63,2,1,'基本权限','redis:manage',1,'null',1,'admin',1,'admin','2021-07-20 10:48:26','2021-07-20 10:48:26'),
	 (71,61,2,1,'数据保存','redis:data:save',6,'null',1,'admin',1,'admin','2021-08-17 11:20:37','2021-08-17 11:20:37'),
	 (72,3,2,1,'终止进程','machine:killprocess',6,'null',1,'admin',1,'admin','2021-08-17 11:20:37','2021-08-17 11:20:37'),
	 (79,0,1,1,'Mongo','/mongo',7,'{"icon":"iconfont icon-mongo","isKeepAlive":true,"routeName":"Mongo"}',1,'admin',1,'admin','2022-05-13 14:00:41','2023-03-16 14:23:22'),
	 (80,79,1,1,'数据操作','mongo-data-operation',1,'{"component":"ops/mongo/MongoDataOp","icon":"iconfont icon-mongo","isKeepAlive":true,"routeName":"MongoDataOp"}',1,'admin',1,'admin','2022-05-13 14:03:58','2023-03-15 17:15:02'),
	 (81,80,2,1,'基本权限','mongo:base',1,'null',1,'admin',1,'admin','2022-05-13 14:04:16','2022-05-13 14:04:16'),
	 (82,79,1,1,'Mongo管理','mongo-manage',2,'{"component":"ops/mongo/MongoList","icon":"iconfont icon-mongo","isKeepAlive":true,"routeName":"MongoList"}',1,'admin',1,'admin','2022-05-16 18:13:06','2023-03-15 17:26:55'),
	 (83,82,2,1,'基本权限','mongo:manage:base',1,'null',1,'admin',1,'admin','2022-05-16 18:13:25','2022-05-16 18:13:25'),
	 (84,4,1,1,'操作日志','syslogs',4,'{"component":"system/syslog/SyslogList","icon":"Tickets","routeName":"SyslogList"}',1,'admin',1,'admin','2022-07-13 19:57:07','2023-03-14 15:44:45'),
	 (85,84,2,1,'操作日志基本权限','syslog',1,'null',1,'admin',1,'admin','2022-07-13 19:57:55','2022-07-13 19:57:55'),
	 (87,4,1,1,'系统配置','configs',5,'{"component":"system/config/ConfigList","icon":"Setting","isKeepAlive":true,"routeName":"ConfigList"}',1,'admin',1,'admin','2022-08-25 22:18:55','2023-03-15 11:06:07'),
	 (88,87,2,1,'基本权限','config:base',1,'null',1,'admin',1,'admin','2022-08-25 22:19:35','2022-08-25 22:19:35'),
	 (93,0,1,1,'标签管理','/tag',3,'{"icon":"CollectionTag","isKeepAlive":true,"routeName":"Tag"}',1,'admin',1,'admin','2022-10-24 15:18:40','2022-10-24 15:24:29'),
	 (94,93,1,1,'标签树','tag-trees',1,'{"component":"ops/tag/TagTreeList","icon":"CollectionTag","isKeepAlive":true,"routeName":"TagTreeList"}',1,'admin',1,'admin','2022-10-24 15:19:40','2023-03-14 14:30:51'),
	 (95,93,1,1,'团队管理','teams',2,'{"component":"ops/tag/TeamList","icon":"UserFilled","isKeepAlive":true,"routeName":"TeamList"}',1,'admin',1,'admin','2022-10-24 15:20:09','2023-03-14 14:31:03'),
	 (96,94,2,1,'保存标签','tag:save',1,'null',1,'admin',1,'admin','2022-10-24 15:20:40','2022-10-26 13:58:36'),
	 (97,95,2,1,'保存团队','team:save',1,'null',1,'admin',1,'admin','2022-10-24 15:20:57','2022-10-26 13:58:56'),
	 (98,94,2,1,'删除标签','tag:del',2,'null',1,'admin',1,'admin','2022-10-26 13:58:47','2022-10-26 13:58:47'),
	 (99,95,2,1,'删除团队','team:del',2,'null',1,'admin',1,'admin','2022-10-26 13:59:06','2022-10-26 13:59:06'),
	 (100,95,2,1,'新增团队成员','team:member:save',3,'null',1,'admin',1,'admin','2022-10-26 13:59:27','2022-10-26 13:59:27'),
	 (101,95,2,1,'移除团队成员','team:member:del',4,'null',1,'admin',1,'admin','2022-10-26 13:59:43','2022-10-26 13:59:43'),
	 (102,95,2,1,'保存团队标签','team:tag:save',5,'null',1,'admin',1,'admin','2022-10-26 13:59:57','2022-10-26 13:59:57'),
	 (103,2,1,1,'授权凭证','authcerts',6,'{"component":"ops/machine/authcert/AuthCertList","icon":"Unlock","isKeepAlive":true,"routeName":"AuthCertList"}',1,'admin',1,'admin','2023-02-23 11:36:26','2023-03-14 14:33:28'),
	 (104,103,2,1,'基本权限','authcert',1,'null',1,'admin',1,'admin','2023-02-23 11:37:24','2023-02-23 11:37:24'),
	 (105,103,2,1,'保存权限','authcert:save',2,'null',1,'admin',1,'admin','2023-02-23 11:37:54','2023-02-23 11:37:54'),
	 (106,103,2,1,'删除权限','authcert:del',3,'null',1,'admin',1,'admin','2023-02-23 11:38:09','2023-02-23 11:38:09'),
	 (108,61,2,1,'数据删除','redis:data:del',3,'null',1,'admin',1,'admin','2023-03-14 17:20:00','2023-03-14 17:20:00'),
	 (109,3,2,1,'关闭连接','machine:close-cli',6,'null',1,'admin',1,'admin','2023-03-16 16:11:04','2023-03-16 16:11:04');
COMMIT;

-- ----------------------------
-- Table structure for t_sys_role
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_role`;
CREATE TABLE `t_sys_role` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '角色code',
  `status` tinyint(255) DEFAULT NULL,
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `type` tinyint(2) NOT NULL COMMENT '类型：1:公共角色；2:特殊角色',
  `create_time` datetime DEFAULT NULL,
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='角色表';

-- ----------------------------
-- Records of t_sys_role
-- ----------------------------
BEGIN;
INSERT INTO `t_sys_role` VALUES (1, '超级管理员', 'SUPBER_ADMIN', 1, '权限超级大，拥有所有权限', 2, '2021-05-27 14:09:50', 1, 'admin', '2021-05-28 10:26:28', 1, 'admin');
INSERT INTO `t_sys_role` VALUES (6, '普通管理员', 'ADMIN', 1, '只拥有部分管理权限', 2, '2021-05-28 15:55:36', 1, 'admin', '2021-05-28 15:55:36', 1, 'admin');
INSERT INTO `t_sys_role` VALUES (7, '公共角色', 'COMMON', 1, '所有账号基础角色', 1, '2021-07-06 15:05:47', 1, 'admin', '2021-07-06 15:05:47', 1, 'admin');
INSERT INTO `t_sys_role` VALUES (8, '开发', 'DEV', 1, '研发人员', 0, '2021-07-09 10:46:10', 1, 'admin', '2021-07-09 10:46:10', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_sys_role_resource
-- ----------------------------
DROP TABLE IF EXISTS `t_sys_role_resource`;
CREATE TABLE `t_sys_role_resource` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `role_id` bigint(20) NOT NULL,
  `resource_id` bigint(20) NOT NULL,
  `creator_id` bigint(20) unsigned DEFAULT NULL,
  `creator` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=526 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='角色资源关联表';

-- ----------------------------
-- Records of t_sys_role_resource
-- ----------------------------
BEGIN;
INSERT INTO `t_sys_role_resource` (id,role_id,resource_id,creator_id,creator,create_time) VALUES
	 (1,1,1,1,'admin','2021-05-27 15:07:39'),
	 (323,1,2,1,'admin','2021-05-28 09:04:50'),
	 (326,1,4,1,'admin','2021-05-28 09:04:50'),
	 (327,1,5,1,'admin','2021-05-28 09:04:50'),
	 (328,1,11,1,'admin','2021-05-28 09:04:50'),
	 (335,1,14,1,'admin','2021-05-28 17:42:21'),
	 (336,1,3,1,'admin','2021-05-28 17:42:43'),
	 (337,1,12,1,'admin','2021-05-28 17:42:43'),
	 (338,6,2,1,'admin','2021-05-28 19:19:38'),
	 (339,6,3,1,'admin','2021-05-28 19:19:38'),
	 (342,6,1,1,'admin','2021-05-29 01:31:22'),
	 (343,5,1,1,'admin','2021-05-31 14:05:23'),
	 (344,5,4,1,'admin','2021-05-31 14:05:23'),
	 (345,5,14,1,'admin','2021-05-31 14:05:23'),
	 (346,5,5,1,'admin','2021-05-31 14:05:23'),
	 (347,5,11,1,'admin','2021-05-31 14:05:23'),
	 (348,5,3,1,'admin','2021-05-31 16:33:14'),
	 (349,5,12,1,'admin','2021-05-31 16:33:14'),
	 (350,5,2,1,'admin','2021-05-31 16:33:14'),
	 (353,1,15,1,'admin','2021-05-31 17:48:33'),
	 (354,1,16,1,'admin','2021-05-31 17:48:33'),
	 (355,1,17,1,'admin','2021-05-31 17:48:33'),
	 (356,1,18,1,'admin','2021-05-31 17:48:33'),
	 (358,1,20,1,'admin','2021-05-31 17:52:08'),
	 (360,1,22,1,'admin','2021-05-31 18:05:04'),
	 (361,1,23,1,'admin','2021-05-31 18:05:04'),
	 (362,1,24,1,'admin','2021-05-31 18:05:04'),
	 (363,1,25,1,'admin','2021-05-31 18:05:04'),
	 (364,1,26,1,'admin','2021-05-31 18:05:04'),
	 (365,1,27,1,'admin','2021-05-31 18:05:04'),
	 (366,1,28,1,'admin','2021-05-31 18:05:04'),
	 (369,1,31,1,'admin','2021-05-31 21:25:56'),
	 (370,1,32,1,'admin','2021-05-31 21:25:56'),
	 (371,1,33,1,'admin','2021-05-31 21:25:56'),
	 (374,1,36,1,'admin','2021-06-01 14:01:57'),
	 (375,1,19,1,'admin','2021-06-01 17:34:03'),
	 (376,1,21,1,'admin','2021-06-01 17:34:03'),
	 (377,1,29,1,'admin','2021-06-01 17:34:03'),
	 (378,1,30,1,'admin','2021-06-01 17:34:03'),
	 (379,1,34,1,'admin','2021-06-01 17:34:03'),
	 (380,1,37,1,'admin','2021-06-03 09:09:42'),
	 (381,1,38,1,'admin','2021-06-03 09:09:42'),
	 (383,1,40,1,'admin','2021-06-08 11:21:52'),
	 (384,1,41,1,'admin','2021-06-08 11:21:52'),
	 (385,1,42,1,'admin','2021-06-08 11:21:52'),
	 (386,1,43,1,'admin','2021-06-08 11:21:52'),
	 (387,1,44,1,'admin','2021-06-08 11:21:52'),
	 (388,1,45,1,'admin','2021-06-08 11:21:52'),
	 (389,1,46,1,'admin','2021-06-08 11:21:52'),
	 (390,1,47,1,'admin','2021-06-08 11:21:52'),
	 (391,6,39,1,'admin','2021-06-08 15:10:58'),
	 (392,6,15,1,'admin','2021-06-08 15:10:58'),
	 (395,6,31,1,'admin','2021-06-08 15:10:58'),
	 (396,6,33,1,'admin','2021-06-08 15:10:58'),
	 (397,6,32,1,'admin','2021-06-08 15:10:58'),
	 (398,6,4,1,'admin','2021-06-08 15:10:58'),
	 (399,6,14,1,'admin','2021-06-08 15:10:58'),
	 (400,6,11,1,'admin','2021-06-08 15:10:58'),
	 (401,6,5,1,'admin','2021-06-08 15:10:58'),
	 (403,7,1,1,'admin','2021-07-06 15:07:09'),
	 (405,1,49,1,'admin','2021-07-07 15:14:17'),
	 (410,1,54,1,'admin','2021-07-08 17:32:19'),
	 (411,1,55,1,'admin','2021-07-08 17:32:19'),
	 (413,1,57,1,'admin','2021-07-09 10:48:50'),
	 (414,1,58,1,'admin','2021-07-09 10:48:50'),
	 (418,8,57,1,'admin','2021-07-09 10:49:46'),
	 (419,8,12,1,'admin','2021-07-09 10:49:46'),
	 (420,8,15,1,'admin','2021-07-09 10:49:46'),
	 (421,8,38,1,'admin','2021-07-09 10:49:46'),
	 (423,8,2,1,'admin','2021-07-09 10:49:46'),
	 (425,8,3,1,'admin','2021-07-09 10:49:46'),
	 (426,8,36,1,'admin','2021-07-09 10:49:46'),
	 (428,1,59,1,'admin','2021-07-09 10:50:20'),
	 (429,8,59,1,'admin','2021-07-09 10:50:32'),
	 (431,6,57,1,'admin','2021-07-12 16:44:12'),
	 (433,1,60,1,'admin','2021-07-19 20:19:29'),
	 (434,1,61,1,'admin','2021-07-19 20:19:29'),
	 (435,1,62,1,'admin','2021-07-19 20:19:29'),
	 (436,1,63,1,'admin','2021-07-20 10:48:39'),
	 (437,1,64,1,'admin','2021-07-20 10:48:39'),
	 (444,7,39,1,'admin','2021-09-09 10:10:30'),
	 (450,6,16,1,'admin','2021-09-09 15:52:38'),
	 (451,6,17,1,'admin','2021-09-09 15:52:38'),
	 (452,6,18,1,'admin','2021-09-09 15:52:38'),
	 (453,6,37,1,'admin','2021-09-09 15:52:38'),
	 (454,6,40,1,'admin','2021-09-09 15:52:38'),
	 (455,6,41,1,'admin','2021-09-09 15:52:38'),
	 (456,6,42,1,'admin','2021-09-09 15:52:38'),
	 (457,6,43,1,'admin','2021-09-09 15:52:38'),
	 (458,6,44,1,'admin','2021-09-09 15:52:38'),
	 (459,6,45,1,'admin','2021-09-09 15:52:38'),
	 (460,6,46,1,'admin','2021-09-09 15:52:38'),
	 (461,6,47,1,'admin','2021-09-09 15:52:38'),
	 (462,6,36,1,'admin','2021-09-09 15:52:38'),
	 (463,6,38,1,'admin','2021-09-09 15:52:38'),
	 (464,6,59,1,'admin','2021-09-09 15:52:38'),
	 (465,6,49,1,'admin','2021-09-09 15:52:38'),
	 (466,6,58,1,'admin','2021-09-09 15:52:38'),
	 (467,6,54,1,'admin','2021-09-09 15:52:38'),
	 (468,6,55,1,'admin','2021-09-09 15:52:38'),
	 (469,6,60,1,'admin','2021-09-09 15:52:38'),
	 (470,6,61,1,'admin','2021-09-09 15:52:38'),
	 (471,6,62,1,'admin','2021-09-09 15:52:38'),
	 (472,6,63,1,'admin','2021-09-09 15:52:38'),
	 (473,6,64,1,'admin','2021-09-09 15:52:38'),
	 (479,6,19,1,'admin','2021-09-09 15:53:56'),
	 (480,6,21,1,'admin','2021-09-09 15:53:56'),
	 (481,6,29,1,'admin','2021-09-09 15:53:56'),
	 (482,6,30,1,'admin','2021-09-09 15:53:56'),
	 (483,6,34,1,'admin','2021-09-09 15:53:56'),
	 (484,6,20,1,'admin','2021-09-09 15:53:56'),
	 (485,6,22,1,'admin','2021-09-09 15:53:56'),
	 (486,6,23,1,'admin','2021-09-09 15:53:56'),
	 (487,6,24,1,'admin','2021-09-09 15:53:56'),
	 (488,6,25,1,'admin','2021-09-09 15:53:56'),
	 (489,6,26,1,'admin','2021-09-09 15:53:56'),
	 (490,6,27,1,'admin','2021-09-09 15:53:56'),
	 (491,6,28,1,'admin','2021-09-09 15:53:56'),
	 (492,8,42,1,'admin','2021-11-05 15:59:16'),
	 (493,8,43,1,'admin','2021-11-05 15:59:16'),
	 (494,8,47,1,'admin','2021-11-05 15:59:16'),
	 (495,8,60,1,'admin','2021-11-05 15:59:16'),
	 (496,8,61,1,'admin','2021-11-05 15:59:16'),
	 (497,8,62,1,'admin','2021-11-05 15:59:16'),
	 (500,1,72,1,'admin','2022-07-14 11:03:09'),
	 (501,1,71,1,'admin','2022-07-14 11:03:09'),
	 (502,1,79,1,'admin','2022-07-14 11:03:09'),
	 (503,1,80,1,'admin','2022-07-14 11:03:09'),
	 (504,1,81,1,'admin','2022-07-14 11:03:09'),
	 (505,1,82,1,'admin','2022-07-14 11:03:09'),
	 (506,1,83,1,'admin','2022-07-14 11:03:09'),
	 (507,1,84,1,'admin','2022-07-14 11:10:11'),
	 (508,1,85,1,'admin','2022-07-14 11:10:11'),
	 (510,1,87,1,'admin','2022-07-14 11:10:11'),
	 (511,1,88,1,'admin','2022-10-08 10:54:06'),
	 (512,8,80,1,'admin','2022-10-08 10:54:34'),
	 (513,8,81,1,'admin','2022-10-08 10:54:34'),
	 (515,8,79,1,'admin','2022-10-08 10:54:34'),
	 (516,1,93,1,'admin','2022-10-26 20:03:14'),
	 (517,1,94,1,'admin','2022-10-26 20:03:14'),
	 (518,1,96,1,'admin','2022-10-26 20:03:14'),
	 (519,1,98,1,'admin','2022-10-26 20:03:14'),
	 (520,1,95,1,'admin','2022-10-26 20:03:14'),
	 (521,1,97,1,'admin','2022-10-26 20:03:14'),
	 (522,1,99,1,'admin','2022-10-26 20:03:14'),
	 (523,1,100,1,'admin','2022-10-26 20:03:14'),
	 (524,1,101,1,'admin','2022-10-26 20:03:14'),
	 (525,1,102,1,'admin','2022-10-26 20:03:14'),
	 (527,1,106,1,'admin','2023-02-23 14:30:54'),
	 (528,1,103,1,'admin','2023-02-23 14:30:54'),
	 (529,1,105,1,'admin','2023-02-23 14:31:00'),
	 (530,1,104,1,'admin','2023-02-24 13:40:26'),
	 (532,1,108,1,'admin','2023-03-14 17:28:06'),
	 (533,6,79,1,'admin','2023-03-14 17:28:50'),
	 (534,6,80,1,'admin','2023-03-14 17:28:50'),
	 (535,6,81,1,'admin','2023-03-14 17:28:50'),
	 (536,6,82,1,'admin','2023-03-14 17:28:50'),
	 (537,6,83,1,'admin','2023-03-14 17:28:50'),
	 (538,6,84,1,'admin','2023-03-14 17:29:00'),
	 (539,6,85,1,'admin','2023-03-14 17:29:00'),
	 (540,6,87,1,'admin','2023-03-14 17:29:00'),
	 (541,6,88,1,'admin','2023-03-14 17:29:00'),
	 (544,1,109,1,'admin','2023-03-16 16:11:25');
COMMIT;

-- ----------------------------
-- Table structure for t_tag_tree
-- ----------------------------
DROP TABLE IF EXISTS `t_tag_tree`;
CREATE TABLE `t_tag_tree` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `pid` bigint(20) NOT NULL DEFAULT '0',
  `code` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '标识符',
  `code_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '标识符路径',
  `name` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '名称',
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_code_path` (`code_path`(100)) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=34 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='标签树';

-- ----------------------------
-- Records of t_tag_tree
-- ----------------------------
BEGIN;
INSERT INTO `t_tag_tree` VALUES (33, 0, 'default', 'default', '默认', '默认标签', '2022-10-26 20:04:19', 1, 'admin', '2022-10-26 20:04:19', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_tag_tree_team
-- ----------------------------
DROP TABLE IF EXISTS `t_tag_tree_team`;
CREATE TABLE `t_tag_tree_team` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `tag_id` bigint(20) NOT NULL COMMENT '项目树id',
  `tag_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `team_id` bigint(20) NOT NULL COMMENT '团队id',
  `create_time` datetime NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tag_id` (`tag_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=32 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='标签树团队关联信息';

-- ----------------------------
-- Records of t_tag_tree_team
-- ----------------------------
BEGIN;
INSERT INTO `t_tag_tree_team` VALUES (31, 33, 'default', 3, '2022-10-26 20:04:45', 1, 'admin', '2022-10-26 20:04:45', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_team
-- ----------------------------
DROP TABLE IF EXISTS `t_team`;
CREATE TABLE `t_team` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '名称',
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '备注',
  `create_time` datetime NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='团队信息';

-- ----------------------------
-- Records of t_team
-- ----------------------------
BEGIN;
INSERT INTO `t_team` VALUES (3, '默认团队', '默认团队', '2022-10-26 20:04:36', 1, 'admin', '2022-10-26 20:04:36', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_team_member
-- ----------------------------
DROP TABLE IF EXISTS `t_team_member`;
CREATE TABLE `t_team_member` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `team_id` bigint(20) NOT NULL COMMENT '项目团队id',
  `account_id` bigint(20) NOT NULL COMMENT '成员id',
  `username` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `create_time` datetime NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  `modifier` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='团队成员表';

-- ----------------------------
-- Records of t_team_member
-- ----------------------------
BEGIN;
INSERT INTO `t_team_member` VALUES (7, 3, 1, 'admin', '2022-10-26 20:04:36', 1, 'admin', '2022-10-26 20:04:36', 1, 'admin');
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;