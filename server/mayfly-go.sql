/*
 Navicat Premium Data Transfer

 Source Server         : self
 Source Server Type    : MySQL
 Source Server Version : 80018
 Source Host           : localhost:3306
 Source Schema         : mayfly-go

 Target Server Type    : MySQL
 Target Server Version : 80018
 File Encoding         : 65001

 Date: 08/06/2021 14:52:42
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for t_account
-- ----------------------------
DROP TABLE IF EXISTS `t_account`;
CREATE TABLE `t_account` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(12) NOT NULL,
  `password` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `status` tinyint(4) DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `creator_id` bigint(255) NOT NULL,
  `creator` varchar(12) NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(255) NOT NULL,
  `modifier` varchar(12) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of t_account
-- ----------------------------
BEGIN;
INSERT INTO `t_account` VALUES (1, 'admin', '7b116d117002fad3f6dde6c718b885b9', 1, '2020-01-01 19:00:00', 1, 'admin', '2020-01-01 19:00:00', 1, 'admin');
INSERT INTO `t_account` VALUES (2, 'admin2', 'e10adc3949ba59abbe56e057f20f883e', -1, '2020-09-09 12:00:09', 2, 'haha', '2020-09-09 12:00:09', 2, 'haha');
COMMIT;

-- ----------------------------
-- Table structure for t_account_role
-- ----------------------------
DROP TABLE IF EXISTS `t_account_role`;
CREATE TABLE `t_account_role` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT 'Id',
  `account_id` bigint(20) NOT NULL COMMENT '账号id',
  `role_id` bigint(20) NOT NULL COMMENT '角色id',
  `creator` varchar(45) CHARACTER SET utf8 COLLATE utf8_general_ci DEFAULT NULL,
  `creator_id` bigint(20) unsigned DEFAULT NULL,
  `create_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=30 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Records of t_account_role
-- ----------------------------
BEGIN;
INSERT INTO `t_account_role` VALUES (23, 3, 6, 'admin', 1, '2021-05-28 16:19:21');
INSERT INTO `t_account_role` VALUES (24, 2, 1, 'admin', 1, '2021-05-28 16:21:38');
INSERT INTO `t_account_role` VALUES (25, 1, 1, 'admin', 1, '2021-05-28 16:21:45');
INSERT INTO `t_account_role` VALUES (26, 4, 6, 'admin', 1, '2021-05-28 17:04:48');
INSERT INTO `t_account_role` VALUES (29, 1, 6, 'admin', 1, '2021-06-02 16:07:44');
COMMIT;

-- ----------------------------
-- Table structure for t_db
-- ----------------------------
DROP TABLE IF EXISTS `t_db`;
CREATE TABLE `t_db` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) DEFAULT NULL COMMENT '数据库名称',
  `host` varchar(20) NOT NULL,
  `port` int(8) NOT NULL,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) DEFAULT NULL,
  `type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '数据库类型(mysql...)',
  `database` varchar(32) DEFAULT NULL,
  `network` varchar(8) DEFAULT NULL,
  `remark` varchar(125) DEFAULT NULL COMMENT '备注，描述等',
  `create_time` datetime DEFAULT NULL,
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(32) DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(32) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='数据库信息表';

-- ----------------------------
-- Records of t_db
-- ----------------------------
BEGIN;
INSERT INTO `t_db` VALUES (1, 'mayfly-go', 'localhost', 3306, 'root', '******', 'mysql', 'mayfly-job', 'tcp', NULL, '2020-12-14 15:04:43', NULL, NULL, '2020-12-14 15:04:48', NULL, NULL);
INSERT INTO `t_db` VALUES (2, 'maylfy', '114.67.67.10', 3306, 'root', '******', 'mysql', 'mayfly', 'tcp', NULL, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO `t_db` VALUES (3, 'mayfly-oracle', 'localhost', 1521, 'TEST', '******', 'godror', 'xe', 'tcp', NULL, NULL, NULL, NULL, NULL, NULL, NULL);
COMMIT;

-- ----------------------------
-- Table structure for t_db_sql
-- ----------------------------
DROP TABLE IF EXISTS `t_db_sql`;
CREATE TABLE `t_db_sql` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `db_id` bigint(20) NOT NULL COMMENT '数据库id',
  `sql` text,
  `type` tinyint(8) NOT NULL,
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(32) DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of t_db_sql
-- ----------------------------
BEGIN;
INSERT INTO `t_db_sql` VALUES (2, 1, 'SELECT\n  *\nFROM\n  t_account\n  \nSELECT\n  *\nFROM\n  t_db\n  \nSELECT\n  *\nFROM\n  t_db_sql', 1, 1, 'admin', '2020-12-18 15:27:53', '2021-04-25 11:09:04', 1, 'admin');
INSERT INTO `t_db_sql` VALUES (3, 2, 'SELECT\n  *\nFROM\n  t_account\n  \nSELECT\n  *\nFROM\n  t_resource\n  \nSELECT\n  *\nFROM\n  t_sys_operation_log\nORDER BY\n  id DESC\nLIMIT\n  20', 1, 1, 'admin', '2020-12-18 17:13:36', '2021-04-21 10:16:21', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_machine
-- ----------------------------
DROP TABLE IF EXISTS `t_machine`;
CREATE TABLE `t_machine` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) DEFAULT NULL,
  `ip` varchar(18) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `port` int(12) NOT NULL,
  `username` varchar(12) NOT NULL,
  `password` varchar(32) DEFAULT NULL,
  `need_monitor` tinyint(2) DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `creator` varchar(16) DEFAULT NULL,
  `creator_id` bigint(32) DEFAULT NULL,
  `update_time` datetime NOT NULL,
  `modifier` varchar(12) DEFAULT NULL,
  `modifier_id` bigint(32) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of t_machine
-- ----------------------------
BEGIN;
INSERT INTO `t_machine` VALUES (1, 'eatlife', '148.70.36.197', 22, 'root', '******', 1, '2020-01-01 10:10:10', NULL, NULL, '2020-01-01 10:10:10', NULL, NULL);
INSERT INTO `t_machine` VALUES (2, 'JD云', '114.67.67.10', 22, 'root', '******', 1, '2020-08-18 10:00:00', NULL, NULL, '2020-08-18 10:00:00', NULL, NULL);
INSERT INTO `t_machine` VALUES (4, '腾讯云', '118.24.26.101', 22, 'root', '******', NULL, '2021-05-10 15:57:49', 'admin', 1, '2021-05-10 15:57:49', 'admin', 1);
COMMIT;

-- ----------------------------
-- Table structure for t_machine_file
-- ----------------------------
DROP TABLE IF EXISTS `t_machine_file`;
CREATE TABLE `t_machine_file` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '机器文件配置（linux一切皆文件，故也可以表示目录）',
  `machine_id` bigint(20) NOT NULL,
  `name` varchar(45) NOT NULL,
  `path` varchar(45) NOT NULL,
  `type` varchar(45) NOT NULL COMMENT '1：目录；2：文件',
  `creator_id` bigint(20) unsigned DEFAULT NULL,
  `creator` varchar(45) CHARACTER SET utf8 COLLATE utf8_general_ci DEFAULT NULL,
  `modifier_id` bigint(20) unsigned DEFAULT NULL,
  `modifier` varchar(45) CHARACTER SET utf8 COLLATE utf8_general_ci DEFAULT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=31 DEFAULT CHARSET=utf8 COMMENT='机器文件';

-- ----------------------------
-- Records of t_machine_file
-- ----------------------------
BEGIN;
INSERT INTO `t_machine_file` VALUES (1, 1, 'redis配置文件', '/etc/my.cnf', '2', 1, 'admin', NULL, NULL, '2019-08-21 11:30:33', NULL);
INSERT INTO `t_machine_file` VALUES (10, 2, 'java-dockerfile', '/usr/local/java/Dockerfile', '2', 1, 'admin', NULL, NULL, '2019-11-06 15:16:12', NULL);
INSERT INTO `t_machine_file` VALUES (11, 1, 'usr', '/usr', '1', 1, 'admin', NULL, NULL, '2019-11-06 15:16:12', NULL);
INSERT INTO `t_machine_file` VALUES (12, 1, 'opt', '/opt', '1', 1, 'admin', NULL, NULL, '2019-11-06 15:16:12', NULL);
INSERT INTO `t_machine_file` VALUES (13, 3, '根目录', '/', '1', 1, 'admin', NULL, NULL, '2019-11-18 05:19:28', NULL);
INSERT INTO `t_machine_file` VALUES (14, 1, 'usr', '/usr/local', '1', 1, 'admin', NULL, NULL, '2019-11-19 05:52:28', NULL);
INSERT INTO `t_machine_file` VALUES (15, 2, 'usr', '/usr/', '1', 1, 'admin', NULL, NULL, '2019-11-19 08:35:19', NULL);
INSERT INTO `t_machine_file` VALUES (16, 3, '软件安装目录', '/usr/local', '1', 1, 'admin', NULL, NULL, '2019-11-19 08:40:55', NULL);
INSERT INTO `t_machine_file` VALUES (23, 4, '根目录', '/', '1', 1, 'admin', NULL, NULL, '2019-12-11 07:54:04', NULL);
INSERT INTO `t_machine_file` VALUES (24, 4, '项目目录', '/usr/local/java', '1', 1, 'admin', NULL, NULL, '2020-03-07 07:12:20', NULL);
INSERT INTO `t_machine_file` VALUES (25, 8, '软件安装位置', '/usr/local', '1', 1, 'admin', NULL, NULL, '2020-08-28 09:41:56', NULL);
INSERT INTO `t_machine_file` VALUES (28, 2, 'java目录', '/usr/local/java', '1', 1, 'admin', 1, 'admin', '2021-05-08 17:32:25', '2021-05-08 17:32:25');
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
  `sys_load` varchar(32) DEFAULT NULL,
  `create_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=6345 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of t_machine_monitor
-- ----------------------------

-- ----------------------------
-- Table structure for t_machine_script
-- ----------------------------
DROP TABLE IF EXISTS `t_machine_script`;
CREATE TABLE `t_machine_script` (
  `id` bigint(64) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '脚本名',
  `machine_id` bigint(64) NOT NULL COMMENT '机器id[0:公共]',
  `script` text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci COMMENT '脚本内容',
  `description` varchar(255) DEFAULT NULL COMMENT '脚本描述',
  `type` tinyint(8) DEFAULT NULL COMMENT '脚本类型[1: 有结果；2：无结果；3：实时交互]',
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(32) DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(255) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of t_machine_script
-- ----------------------------
BEGIN;
INSERT INTO `t_machine_script` VALUES (1, 'sys_info', 9999999, '# 获取系统cpu信息\nfunction get_cpu_info() {\n  Physical_CPUs=$(grep \"physical id\" /proc/cpuinfo | sort | uniq | wc -l)\n  Virt_CPUs=$(grep \"processor\" /proc/cpuinfo | wc -l)\n  CPU_Kernels=$(grep \"cores\" /proc/cpuinfo | uniq | awk -F \': \' \'{print $2}\')\n  CPU_Type=$(grep \"model name\" /proc/cpuinfo | awk -F \': \' \'{print $2}\' | sort | uniq)\n  CPU_Arch=$(uname -m)\n  echo -e \'\\n-------------------------- CPU信息 --------------------------\'\n  cat <<EOF | column -t\n物理CPU个数: $Physical_CPUs\n逻辑CPU个数: $Virt_CPUs\n每CPU核心数: $CPU_Kernels\nCPU型号: $CPU_Type\nCPU架构: $CPU_Arch\nEOF\n}\n\n# 获取系统信息\nfunction get_systatus_info() {\n  sys_os=$(uname -o)\n  sys_release=$(cat /etc/redhat-release)\n  sys_kernel=$(uname -r)\n  sys_hostname=$(hostname)\n  sys_selinux=$(getenforce)\n  sys_lang=$(echo $LANG)\n  sys_lastreboot=$(who -b | awk \'{print $3,$4}\')\n  echo -e \'-------------------------- 系统信息 --------------------------\'\n  cat <<EOF | column -t\n系统: ${sys_os}\n发行版本:   ${sys_release}\n系统内核:   ${sys_kernel}\n主机名:    ${sys_hostname}\nselinux状态:  ${sys_selinux}\n系统语言:   ${sys_lang}\n系统最后重启时间:   ${sys_lastreboot}\nEOF\n}\n\nget_systatus_info\n#echo -e \"\\n\"\nget_cpu_info', '获取系统信息', 1, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO `t_machine_script` VALUES (2, 'get_process_by_name', 9999999, '#! /bin/bash\n# Function: 根据输入的程序的名字过滤出所对应的PID，并显示出详细信息，如果有几个PID，则全部显示\nNAME=java\nN=`ps -aux | grep $NAME | grep -v grep | wc -l`    ##统计进程总数\nif [ $N -le 0 ];then\n  echo \"无该进程！\"\nfi\ni=1\nwhile [ $N -gt 0 ]\ndo\n  echo \"进程PID: `ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $2}\'`\"\n  echo \"进程命令：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $11}\'`\"\n  echo \"进程所属用户: `ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $1}\'`\"\n  echo \"CPU占用率：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $3}\'`%\"\n  echo \"内存占用率：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $4}\'`%\"\n  echo \"进程开始运行的时刻：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $9}\'`\"\n  echo \"进程运行的时间：`  ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $11}\'`\"\n  echo \"进程状态：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $8}\'`\"\n  echo \"进程虚拟内存：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $5}\'`\"\n  echo \"进程共享内存：`ps -aux | grep $NAME | grep -v grep | awk \'NR==\'$i\'{print $0}\'| awk \'{print $6}\'`\"\n  echo \"***************************************************************\"\n  let N-- i++\ndone', '获取进程运行状态', 1, NULL, NULL, 1, 'admin', NULL, '2021-05-05 16:52:54');
INSERT INTO `t_machine_script` VALUES (3, 'sys_run_info', 9999999, '#!/bin/bash\n# 获取要监控的本地服务器IP地址\nIP=`ifconfig | grep inet | grep -vE \'inet6|127.0.0.1\' | awk \'{print $2}\'`\necho \"IP地址：\"$IP\n \n# 获取cpu总核数\ncpu_num=`grep -c \"model name\" /proc/cpuinfo`\necho \"cpu总核数：\"$cpu_num\n \n# 1、获取CPU利用率\n################################################\n#us 用户空间占用CPU百分比\n#sy 内核空间占用CPU百分比\n#ni 用户进程空间内改变过优先级的进程占用CPU百分比\n#id 空闲CPU百分比\n#wa 等待输入输出的CPU时间百分比\n#hi 硬件中断\n#si 软件中断\n#################################################\n# 获取用户空间占用CPU百分比\ncpu_user=`top -b -n 1 | grep Cpu | awk \'{print $2}\' | cut -f 1 -d \"%\"`\necho \"用户空间占用CPU百分比：\"$cpu_user\n \n# 获取内核空间占用CPU百分比\ncpu_system=`top -b -n 1 | grep Cpu | awk \'{print $4}\' | cut -f 1 -d \"%\"`\necho \"内核空间占用CPU百分比：\"$cpu_system\n \n# 获取空闲CPU百分比\ncpu_idle=`top -b -n 1 | grep Cpu | awk \'{print $8}\' | cut -f 1 -d \"%\"`\necho \"空闲CPU百分比：\"$cpu_idle\n \n# 获取等待输入输出占CPU百分比\ncpu_iowait=`top -b -n 1 | grep Cpu | awk \'{print $10}\' | cut -f 1 -d \"%\"`\necho \"等待输入输出占CPU百分比：\"$cpu_iowait\n \n#2、获取CPU上下文切换和中断次数\n# 获取CPU中断次数\ncpu_interrupt=`vmstat -n 1 1 | sed -n 3p | awk \'{print $11}\'`\necho \"CPU中断次数：\"$cpu_interrupt\n \n# 获取CPU上下文切换次数\ncpu_context_switch=`vmstat -n 1 1 | sed -n 3p | awk \'{print $12}\'`\necho \"CPU上下文切换次数：\"$cpu_context_switch\n \n#3、获取CPU负载信息\n# 获取CPU15分钟前到现在的负载平均值\ncpu_load_15min=`uptime | awk \'{print $11}\' | cut -f 1 -d \',\'`\necho \"CPU 15分钟前到现在的负载平均值：\"$cpu_load_15min\n \n# 获取CPU5分钟前到现在的负载平均值\ncpu_load_5min=`uptime | awk \'{print $10}\' | cut -f 1 -d \',\'`\necho \"CPU 5分钟前到现在的负载平均值：\"$cpu_load_5min\n \n# 获取CPU1分钟前到现在的负载平均值\ncpu_load_1min=`uptime | awk \'{print $9}\' | cut -f 1 -d \',\'`\necho \"CPU 1分钟前到现在的负载平均值：\"$cpu_load_1min\n \n# 获取任务队列(就绪状态等待的进程数)\ncpu_task_length=`vmstat -n 1 1 | sed -n 3p | awk \'{print $1}\'`\necho \"CPU任务队列长度：\"$cpu_task_length\n \n#4、获取内存信息\n# 获取物理内存总量\nmem_total=`free -h | grep Mem | awk \'{print $2}\'`\necho \"物理内存总量：\"$mem_total\n \n# 获取操作系统已使用内存总量\nmem_sys_used=`free -h | grep Mem | awk \'{print $3}\'`\necho \"已使用内存总量(操作系统)：\"$mem_sys_used\n \n# 获取操作系统未使用内存总量\nmem_sys_free=`free -h | grep Mem | awk \'{print $4}\'`\necho \"剩余内存总量(操作系统)：\"$mem_sys_free\n \n# 获取应用程序已使用的内存总量\nmem_user_used=`free | sed -n 3p | awk \'{print $3}\'`\necho \"已使用内存总量(应用程序)：\"$mem_user_used\n \n# 获取应用程序未使用内存总量\nmem_user_free=`free | sed -n 3p | awk \'{print $4}\'`\necho \"剩余内存总量(应用程序)：\"$mem_user_free\n \n# 获取交换分区总大小\nmem_swap_total=`free | grep Swap | awk \'{print $2}\'`\necho \"交换分区总大小：\"$mem_swap_total\n \n# 获取已使用交换分区大小\nmem_swap_used=`free | grep Swap | awk \'{print $3}\'`\necho \"已使用交换分区大小：\"$mem_swap_used\n \n# 获取剩余交换分区大小\nmem_swap_free=`free | grep Swap | awk \'{print $4}\'`\necho \"剩余交换分区大小：\"$mem_swap_free', '获取cpu、内存等系统运行状态', 1, NULL, NULL, NULL, NULL, NULL, '2021-04-25 15:07:16');
INSERT INTO `t_machine_script` VALUES (4, 'top', 9999999, 'top', '实时获取系统运行状态', 3, NULL, NULL, 1, 'admin', NULL, '2021-05-24 15:58:20');
INSERT INTO `t_machine_script` VALUES (9, 'fasf', 0, 'fafsd', 'fasdfa', 2, 1, 'admin', 1, 'admin', '2021-04-25 15:55:25', '2021-04-25 15:55:25');
INSERT INTO `t_machine_script` VALUES (10, 'fas', 0, 'd', 'f', 2, 1, 'admin', 1, 'admin', '2021-04-25 15:58:14', '2021-04-25 15:58:14');
INSERT INTO `t_machine_script` VALUES (11, 'tests', 0, 'fsfsdf', 'fsdfs', 2, 1, 'admin', 1, 'admin', '2021-04-25 16:13:06', '2021-04-25 16:13:06');
INSERT INTO `t_machine_script` VALUES (12, 'fafas', 0, 'sdsd', 'fsfsd', 2, 1, 'admin', 1, 'admin', '2021-04-25 16:14:41', '2021-04-25 16:14:41');
INSERT INTO `t_machine_script` VALUES (15, 'mvn_install', 4, 'mvn clean -Dmaven.test.skip=true install', 'mvn打包', 3, 1, 'admin', 1, 'admin', '2021-05-10 15:58:34', '2021-06-03 11:12:25');
COMMIT;

-- ----------------------------
-- Table structure for t_resource
-- ----------------------------
DROP TABLE IF EXISTS `t_resource`;
CREATE TABLE `t_resource` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `pid` int(11) NOT NULL,
  `type` tinyint(255) NOT NULL COMMENT '1：菜单路由；2：资源（按钮等）',
  `status` int(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '主要用于按钮等资源',
  `weight` int(11) DEFAULT NULL,
  `meta` varchar(255) DEFAULT NULL COMMENT '原数据',
  `creator_id` bigint(20) NOT NULL,
  `creator` varchar(255) NOT NULL,
  `modifier_id` bigint(20) NOT NULL,
  `modifier` varchar(255) NOT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=48 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of t_resource
-- ----------------------------
BEGIN;
INSERT INTO `t_resource` VALUES (1, 0, 1, 1, '首页', '/home', 1, '{\"component\":\"Home\",\"icon\":\"el-icon-s-home\",\"isAffix\":true,\"isKeepAlive\":true,\"routeName\":\"Home\"}', 1, 'admin', 1, 'admin', '2021-05-25 16:44:41', '2021-05-27 09:12:56');
INSERT INTO `t_resource` VALUES (2, 0, 1, 1, '运维', '/ops', 3, '{\"icon\":\"el-icon-monitor\",\"isKeepAlive\":true,\"redirect\":\"machine/list\",\"routeName\":\"Ops\"}', 1, 'admin', 1, 'admin', '2021-05-25 16:48:16', '2021-06-08 14:20:24');
INSERT INTO `t_resource` VALUES (3, 2, 1, 1, '机器列表', 'machines', 1, '{\"component\":\"MachineList\",\"icon\":\"el-icon-menu\",\"isKeepAlive\":true,\"routeName\":\"MachineList\"}', 2, 'admin', 1, 'admin', '2021-05-25 16:50:04', '2021-06-03 10:03:29');
INSERT INTO `t_resource` VALUES (4, 0, 1, 1, '系统管理', '/sys', 4, '{\"icon\":\"el-icon-setting\",\"isKeepAlive\":true,\"redirect\":\"/sys/resources\",\"routeName\":\"sys\"}', 1, 'admin', 1, 'admin', '2021-05-26 15:20:20', '2021-06-08 14:20:34');
INSERT INTO `t_resource` VALUES (5, 4, 1, 1, '资源管理', 'resources', 3, '{\"component\":\"ResourceList\",\"icon\":\"el-icon-menu\",\"isKeepAlive\":true,\"routeName\":\"ResourceList\"}', 1, 'admin', 1, 'admin', '2021-05-26 15:23:07', '2021-06-08 11:27:55');
INSERT INTO `t_resource` VALUES (9, 0, 1, 1, 'iframes', '/iframes', 4, '{\"component\":\"RouterParent\",\"icon\":\"el-icon-pear\",\"isIframe\":true,\"isKeepAlive\":true,\"link\":\"https://www.baidu.com\",\"routeName\":\"Iframe\"}', 1, 'admin', 1, 'admin', '2021-05-27 09:58:37', '2021-06-02 11:23:23');
INSERT INTO `t_resource` VALUES (11, 4, 1, 1, '角色管理', 'roles', 2, '{\"component\":\"RoleList\",\"icon\":\"el-icon-menu\",\"isKeepAlive\":true,\"routeName\":\"RoleList\"}', 1, 'admin', 1, 'admin', '2021-05-27 11:15:35', '2021-06-03 09:59:41');
INSERT INTO `t_resource` VALUES (12, 3, 2, 1, '机器终端按钮', 'machine:terminal', 4, '', 1, 'admin', 1, 'admin', '2021-05-28 14:06:02', '2021-05-31 17:47:59');
INSERT INTO `t_resource` VALUES (14, 4, 1, 1, '账号管理', 'accounts', 1, '{\"component\":\"AccountList\",\"icon\":\"el-icon-menu\",\"isKeepAlive\":true,\"routeName\":\"AccountList\"}', 1, 'admin', 1, 'admin', '2021-05-28 14:56:25', '2021-06-03 09:39:22');
INSERT INTO `t_resource` VALUES (15, 3, 2, 1, '文件管理按钮', 'machine:file', 5, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:44:37', '2021-05-31 17:48:07');
INSERT INTO `t_resource` VALUES (16, 3, 2, 1, '机器添加按钮', 'machine:add', 1, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:46:11', '2021-05-31 19:34:15');
INSERT INTO `t_resource` VALUES (17, 3, 2, 1, '机器编辑按钮', 'machine:update', 2, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:46:23', '2021-05-31 19:34:18');
INSERT INTO `t_resource` VALUES (18, 3, 2, 1, '机器删除按钮', 'machine:del', 3, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:46:36', '2021-05-31 19:34:17');
INSERT INTO `t_resource` VALUES (19, 14, 2, 1, '角色分配按钮', 'account:saveRoles', 1, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:50:51', '2021-05-31 19:19:30');
INSERT INTO `t_resource` VALUES (20, 11, 2, 1, '分配菜单&权限按钮', 'role:saveResources', 1, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:51:41', '2021-05-31 19:33:37');
INSERT INTO `t_resource` VALUES (21, 14, 2, 1, '账号删除按钮', 'account:del', 2, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:02:01', '2021-05-31 18:02:01');
INSERT INTO `t_resource` VALUES (22, 11, 2, 1, '角色删除按钮', 'role:del', 2, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:02:29', '2021-05-31 19:33:38');
INSERT INTO `t_resource` VALUES (23, 11, 2, 1, '角色新增按钮', 'role:add', 3, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:02:44', '2021-05-31 19:33:39');
INSERT INTO `t_resource` VALUES (24, 11, 2, 1, '角色编辑按钮', 'role:update', 4, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:02:57', '2021-05-31 19:33:40');
INSERT INTO `t_resource` VALUES (25, 5, 2, 1, '资源新增按钮', 'resource:add', 1, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:03:33', '2021-05-31 19:31:47');
INSERT INTO `t_resource` VALUES (26, 5, 2, 1, '资源删除按钮', 'resource:del', 2, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:03:47', '2021-05-31 19:29:40');
INSERT INTO `t_resource` VALUES (27, 5, 2, 1, '资源编辑按钮', 'resource:update', 3, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:04:03', '2021-05-31 19:29:40');
INSERT INTO `t_resource` VALUES (28, 5, 2, 1, '资源禁用启用按钮', 'resource:changeStatus', 4, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:04:33', '2021-05-31 18:04:33');
INSERT INTO `t_resource` VALUES (29, 14, 2, 1, '账号添加按钮', 'account:add', 3, NULL, 1, 'admin', 1, 'admin', '2021-05-31 19:23:42', '2021-05-31 19:23:42');
INSERT INTO `t_resource` VALUES (30, 14, 2, 1, '账号编辑修改按钮', 'account:update', 4, NULL, 1, 'admin', 1, 'admin', '2021-05-31 19:23:58', '2021-05-31 19:23:58');
INSERT INTO `t_resource` VALUES (31, 14, 2, 1, '账号管理基本权限', 'account', 0, NULL, 1, 'admin', 1, 'admin', '2021-05-31 21:25:06', '2021-05-31 21:25:06');
INSERT INTO `t_resource` VALUES (32, 5, 2, 1, '资源管理基本权限', 'resource', 0, NULL, 1, 'admin', 1, 'admin', '2021-05-31 21:25:25', '2021-05-31 21:25:25');
INSERT INTO `t_resource` VALUES (33, 11, 2, 1, '角色管理基本权限', 'role', 0, NULL, 1, 'admin', 1, 'admin', '2021-05-31 21:25:40', '2021-05-31 21:25:40');
INSERT INTO `t_resource` VALUES (34, 14, 2, 1, '账号启用禁用按钮', 'account:changeStatus', 5, NULL, 1, 'admin', 1, 'admin', '2021-05-31 21:29:48', '2021-05-31 21:29:48');
INSERT INTO `t_resource` VALUES (36, 2, 1, 1, 'DBMS', 'dbms', 2, '{\"icon\":\"el-icon-date\",\"isKeepAlive\":true,\"routeName\":\"DBMS\"}', 1, 'admin', 1, 'admin', '2021-06-01 14:01:33', '2021-06-03 09:58:54');
INSERT INTO `t_resource` VALUES (37, 3, 2, 1, '添加文件配置', 'machine:addFile', 6, 'null', 1, 'admin', 1, 'admin', '2021-06-01 19:54:23', '2021-06-01 19:54:23');
INSERT INTO `t_resource` VALUES (38, 36, 1, 1, '数据查询', 'select-data', 1, '{\"component\":\"SelectData\",\"icon\":\"el-icon-search\",\"isKeepAlive\":true,\"routeName\":\"SelectData\"}', 1, 'admin', 1, 'admin', '2021-06-03 09:09:29', '2021-06-03 14:34:01');
INSERT INTO `t_resource` VALUES (39, 0, 1, 1, '个人中心', '/personal', 2, '{\"component\":\"Personal\",\"icon\":\"el-icon-menu\",\"isHide\":true,\"isKeepAlive\":true,\"routeName\":\"Personal\"}', 1, 'admin', 1, 'admin', '2021-06-03 14:25:35', '2021-06-03 14:26:03');
INSERT INTO `t_resource` VALUES (40, 3, 2, 1, '文件管理-新增按钮', 'machine:file:add', 7, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:06:26', '2021-06-08 11:12:28');
INSERT INTO `t_resource` VALUES (41, 3, 2, 1, '文件管理-删除按钮', 'machine:file:del', 8, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:06:49', '2021-06-08 11:06:49');
INSERT INTO `t_resource` VALUES (42, 3, 2, 1, '文件管理-写入or下载文件权限', 'machine:file:write', 9, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:07:27', '2021-06-08 11:07:27');
INSERT INTO `t_resource` VALUES (43, 3, 2, 1, '文件管理-文件上传按钮', 'machine:file:upload', 10, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:07:42', '2021-06-08 11:07:42');
INSERT INTO `t_resource` VALUES (44, 3, 2, 1, '文件管理-删除文件按钮', 'machine:file:rm', 11, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:08:12', '2021-06-08 11:08:12');
INSERT INTO `t_resource` VALUES (45, 3, 2, 1, '脚本管理-保存脚本按钮', 'machine:script:save', 12, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:09:01', '2021-06-08 11:09:01');
INSERT INTO `t_resource` VALUES (46, 3, 2, 1, '脚本管理-删除按钮', 'machine:script:del', 13, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:09:27', '2021-06-08 11:09:27');
INSERT INTO `t_resource` VALUES (47, 3, 2, 1, '脚本管理-执行按钮', 'machine:script:run', 14, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:09:50', '2021-06-08 11:09:50');
COMMIT;

-- ----------------------------
-- Table structure for t_role
-- ----------------------------
DROP TABLE IF EXISTS `t_role`;
CREATE TABLE `t_role` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(16) NOT NULL,
  `status` tinyint(255) DEFAULT NULL,
  `remark` varchar(255) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `creator_id` bigint(20) DEFAULT NULL,
  `creator` varchar(16) DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  `modifier_id` bigint(20) DEFAULT NULL,
  `modifier` varchar(16) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of t_role
-- ----------------------------
BEGIN;
INSERT INTO `t_role` VALUES (1, '超级管理员', 1, '权限超级大，拥有所有权限', '2021-05-27 14:09:50', 1, 'admin', '2021-05-28 10:26:28', 1, 'admin');
INSERT INTO `t_role` VALUES (6, '普通管理员', 1, '只拥有部分管理权限', '2021-05-28 15:55:36', 1, 'admin', '2021-05-28 15:55:36', 1, 'admin');
COMMIT;

-- ----------------------------
-- Table structure for t_role_resource
-- ----------------------------
DROP TABLE IF EXISTS `t_role_resource`;
CREATE TABLE `t_role_resource` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `role_id` bigint(20) NOT NULL,
  `resource_id` bigint(20) NOT NULL,
  `creator_id` bigint(20) unsigned DEFAULT NULL,
  `creator` varchar(45) CHARACTER SET utf8 COLLATE utf8_general_ci DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=391 DEFAULT CHARSET=utf8 COMMENT='角色资源表';

-- ----------------------------
-- Records of t_role_resource
-- ----------------------------
BEGIN;
INSERT INTO `t_role_resource` VALUES (1, 1, 1, 1, 'admin', '2021-05-27 15:07:39');
INSERT INTO `t_role_resource` VALUES (323, 1, 2, 1, 'admin', '2021-05-28 09:04:50');
INSERT INTO `t_role_resource` VALUES (326, 1, 4, 1, 'admin', '2021-05-28 09:04:50');
INSERT INTO `t_role_resource` VALUES (327, 1, 5, 1, 'admin', '2021-05-28 09:04:50');
INSERT INTO `t_role_resource` VALUES (328, 1, 11, 1, 'admin', '2021-05-28 09:04:50');
INSERT INTO `t_role_resource` VALUES (335, 1, 14, 1, 'admin', '2021-05-28 17:42:21');
INSERT INTO `t_role_resource` VALUES (336, 1, 3, 1, 'admin', '2021-05-28 17:42:43');
INSERT INTO `t_role_resource` VALUES (337, 1, 12, 1, 'admin', '2021-05-28 17:42:43');
INSERT INTO `t_role_resource` VALUES (338, 6, 2, 1, 'admin', '2021-05-28 19:19:38');
INSERT INTO `t_role_resource` VALUES (339, 6, 3, 1, 'admin', '2021-05-28 19:19:38');
INSERT INTO `t_role_resource` VALUES (340, 6, 12, 1, 'admin', '2021-05-28 19:19:38');
INSERT INTO `t_role_resource` VALUES (342, 6, 1, 1, 'admin', '2021-05-29 01:31:22');
INSERT INTO `t_role_resource` VALUES (343, 5, 1, 1, 'admin', '2021-05-31 14:05:23');
INSERT INTO `t_role_resource` VALUES (344, 5, 4, 1, 'admin', '2021-05-31 14:05:23');
INSERT INTO `t_role_resource` VALUES (345, 5, 14, 1, 'admin', '2021-05-31 14:05:23');
INSERT INTO `t_role_resource` VALUES (346, 5, 5, 1, 'admin', '2021-05-31 14:05:23');
INSERT INTO `t_role_resource` VALUES (347, 5, 11, 1, 'admin', '2021-05-31 14:05:23');
INSERT INTO `t_role_resource` VALUES (348, 5, 3, 1, 'admin', '2021-05-31 16:33:14');
INSERT INTO `t_role_resource` VALUES (349, 5, 12, 1, 'admin', '2021-05-31 16:33:14');
INSERT INTO `t_role_resource` VALUES (350, 5, 2, 1, 'admin', '2021-05-31 16:33:14');
INSERT INTO `t_role_resource` VALUES (353, 1, 15, 1, 'admin', '2021-05-31 17:48:33');
INSERT INTO `t_role_resource` VALUES (354, 1, 16, 1, 'admin', '2021-05-31 17:48:33');
INSERT INTO `t_role_resource` VALUES (355, 1, 17, 1, 'admin', '2021-05-31 17:48:33');
INSERT INTO `t_role_resource` VALUES (356, 1, 18, 1, 'admin', '2021-05-31 17:48:33');
INSERT INTO `t_role_resource` VALUES (358, 1, 20, 1, 'admin', '2021-05-31 17:52:08');
INSERT INTO `t_role_resource` VALUES (360, 1, 22, 1, 'admin', '2021-05-31 18:05:04');
INSERT INTO `t_role_resource` VALUES (361, 1, 23, 1, 'admin', '2021-05-31 18:05:04');
INSERT INTO `t_role_resource` VALUES (362, 1, 24, 1, 'admin', '2021-05-31 18:05:04');
INSERT INTO `t_role_resource` VALUES (363, 1, 25, 1, 'admin', '2021-05-31 18:05:04');
INSERT INTO `t_role_resource` VALUES (364, 1, 26, 1, 'admin', '2021-05-31 18:05:04');
INSERT INTO `t_role_resource` VALUES (365, 1, 27, 1, 'admin', '2021-05-31 18:05:04');
INSERT INTO `t_role_resource` VALUES (366, 1, 28, 1, 'admin', '2021-05-31 18:05:04');
INSERT INTO `t_role_resource` VALUES (369, 1, 31, 1, 'admin', '2021-05-31 21:25:56');
INSERT INTO `t_role_resource` VALUES (370, 1, 32, 1, 'admin', '2021-05-31 21:25:56');
INSERT INTO `t_role_resource` VALUES (371, 1, 33, 1, 'admin', '2021-05-31 21:25:56');
INSERT INTO `t_role_resource` VALUES (374, 1, 36, 1, 'admin', '2021-06-01 14:01:57');
INSERT INTO `t_role_resource` VALUES (375, 1, 19, 1, 'admin', '2021-06-01 17:34:03');
INSERT INTO `t_role_resource` VALUES (376, 1, 21, 1, 'admin', '2021-06-01 17:34:03');
INSERT INTO `t_role_resource` VALUES (377, 1, 29, 1, 'admin', '2021-06-01 17:34:03');
INSERT INTO `t_role_resource` VALUES (378, 1, 30, 1, 'admin', '2021-06-01 17:34:03');
INSERT INTO `t_role_resource` VALUES (379, 1, 34, 1, 'admin', '2021-06-01 17:34:03');
INSERT INTO `t_role_resource` VALUES (380, 1, 37, 1, 'admin', '2021-06-03 09:09:42');
INSERT INTO `t_role_resource` VALUES (381, 1, 38, 1, 'admin', '2021-06-03 09:09:42');
INSERT INTO `t_role_resource` VALUES (382, 1, 39, 1, 'admin', '2021-06-03 14:26:42');
INSERT INTO `t_role_resource` VALUES (383, 1, 40, 1, 'admin', '2021-06-08 11:21:52');
INSERT INTO `t_role_resource` VALUES (384, 1, 41, 1, 'admin', '2021-06-08 11:21:52');
INSERT INTO `t_role_resource` VALUES (385, 1, 42, 1, 'admin', '2021-06-08 11:21:52');
INSERT INTO `t_role_resource` VALUES (386, 1, 43, 1, 'admin', '2021-06-08 11:21:52');
INSERT INTO `t_role_resource` VALUES (387, 1, 44, 1, 'admin', '2021-06-08 11:21:52');
INSERT INTO `t_role_resource` VALUES (388, 1, 45, 1, 'admin', '2021-06-08 11:21:52');
INSERT INTO `t_role_resource` VALUES (389, 1, 46, 1, 'admin', '2021-06-08 11:21:52');
INSERT INTO `t_role_resource` VALUES (390, 1, 47, 1, 'admin', '2021-06-08 11:21:52');
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
