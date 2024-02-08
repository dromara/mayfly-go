--
-- Text encoding used: UTF-8
--
PRAGMA foreign_keys = off;
BEGIN TRANSACTION;

-- Table: t_auth_cert
CREATE TABLE IF NOT EXISTS "t_auth_cert" (
  "id" integer NOT NULL,
  "name" text(32),
  "auth_method" integer(4) NOT NULL,
  "password" text(4200),
  "passphrase" text(32),
  "remark" text(255),
  "create_time"  datetime NOT NULL,
  "creator" text(16) NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier" text(12) NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db
CREATE TABLE IF NOT EXISTS "t_db" (
  "id" integer NOT NULL,
  "code" text(32),
  "name" text(32),
  "database" text(1000),
  "remark" text(125),
  "instance_id" integer(20) NOT NULL,
  "create_time"  datetime,
  "creator_id" integer(20),
  "creator" text(32),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(32),
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_backup
CREATE TABLE IF NOT EXISTS "t_db_backup" (
  "id" integer NOT NULL,
  "name" text(32) NOT NULL,
  "db_instance_id" integer(20) NOT NULL,
  "db_name" text(64) NOT NULL,
  "repeated" integer(1),
  "interval" integer(20),
  "max_save_days" integer(8) NOT NULL DEFAULT '0',
  "start_time"  datetime,
  "enabled" integer(1),
  "enabled_desc" text(64),
  "last_status" integer(4),
  "last_result" text(256),
  "last_time"  datetime,
  "create_time"  datetime,
  "creator_id" integer(20),
  "creator" text(32),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(32),
  "is_deleted" integer(1) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_backup_history
CREATE TABLE IF NOT EXISTS "t_db_backup_history" (
  "id" integer NOT NULL,
  "name" text(64) NOT NULL,
  "db_backup_id" integer(20) NOT NULL,
  "db_instance_id" integer(20) NOT NULL,
  "db_name" text(64) NOT NULL,
  "uuid" text(36) NOT NULL,
  "binlog_file_name" text(32),
  "binlog_sequence" integer(20),
  "binlog_position" integer(20),
  "create_time"  datetime,
  "is_deleted" integer(1) NOT NULL,
  "delete_time"  datetime,
  "restoring" integer(1) NOT NULL DEFAULT '0',
  "deleting" integer(1) NOT NULL DEFAULT '0',
  PRIMARY KEY ("id")
);

-- Table: t_db_binlog
CREATE TABLE IF NOT EXISTS "t_db_binlog" (
  "id" integer NOT NULL,
  "db_instance_id" integer(20) NOT NULL,
  "last_status" integer(20),
  "last_result" text(256),
  "last_time"  datetime,
  "create_time"  datetime,
  "creator_id" integer(20),
  "creator" text(32),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(32),
  "is_deleted" integer(1) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_binlog_history
CREATE TABLE IF NOT EXISTS "t_db_binlog_history" (
  "id" integer NOT NULL,
  "db_instance_id" integer(20) NOT NULL,
  "file_name" text(32),
  "file_size" integer(20),
  "sequence" integer(20),
  "first_event_time"  datetime,
  "last_event_time"  datetime,
  "create_time"  datetime,
  "is_deleted" integer(4) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_data_sync_log
CREATE TABLE IF NOT EXISTS "t_db_data_sync_log" (
  "id" integer NOT NULL,
  "task_id" integer(20) NOT NULL,
  "create_time"  datetime NOT NULL,
  "data_sql_full" text NOT NULL,
  "res_num" integer(11),
  "err_text" text,
  "status" integer(4) NOT NULL,
  "is_deleted" integer(1) NOT NULL,
  PRIMARY KEY ("id")
);

-- Table: t_db_data_sync_task
CREATE TABLE IF NOT EXISTS "t_db_data_sync_task" (
  "id" integer NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(100) NOT NULL,
  "create_time"  datetime NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier" text(100) NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "task_name" text(500) NOT NULL,
  "task_cron" text(50) NOT NULL,
  "src_db_id" integer(20) NOT NULL,
  "src_db_name" text(100),
  "src_tag_path" text(200),
  "target_db_id" integer(20) NOT NULL,
  "target_db_name" text(100),
  "target_tag_path" text(200),
  "target_table_name" text(100),
  "data_sql" text NOT NULL,
  "page_size" integer(11) NOT NULL,
  "upd_field" text(100) NOT NULL,
  "upd_field_val" text(100),
  "id_rule" integer(2) NOT NULL,
  "pk_field" text(100),
  "field_map" text,
  "is_deleted" integer(8),
  "delete_time"  datetime,
  "status" integer(1) NOT NULL,
  "recent_state" integer(1) NOT NULL,
  "task_key" text(100),
  "running_state" integer(1),
  PRIMARY KEY ("id")
);

-- Table: t_db_instance
CREATE TABLE IF NOT EXISTS "t_db_instance" (
  "id" integer NOT NULL,
  "name" text(32),
  "host" text(100) NOT NULL,
  "port" integer(8) NOT NULL,
  "sid" text(255) NOT NULL,
  "username" text(255) NOT NULL,
  "password" text(255),
  "type" text(20) NOT NULL,
  "params" text(125),
  "network" text(20),
  "ssh_tunnel_machine_id" integer(20),
  "remark" text(125),
  "create_time"  datetime,
  "creator_id" integer(20),
  "creator" text(32),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(32),
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_restore
CREATE TABLE IF NOT EXISTS "t_db_restore" (
  "id" integer NOT NULL,
  "db_instance_id" integer(20) NOT NULL,
  "db_name" text(64) NOT NULL,
  "repeated" integer(1),
  "interval" integer(20),
  "start_time"  datetime,
  "enabled" integer(1),
  "enabled_desc" text(64),
  "last_status" integer(4),
  "last_result" text(256),
  "last_time"  datetime,
  "point_in_time"  datetime,
  "db_backup_id" integer(20),
  "db_backup_history_id" integer(20),
  "db_backup_history_name" text(64),
  "create_time"  datetime,
  "creator_id" integer(20),
  "creator" text(32),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(32),
  "is_deleted" integer(1) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_restore_history
CREATE TABLE IF NOT EXISTS "t_db_restore_history" (
  "id" integer NOT NULL,
  "db_restore_id" integer(20) NOT NULL,
  "create_time"  datetime,
  "is_deleted" integer(4) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_sql
CREATE TABLE IF NOT EXISTS "t_db_sql" (
  "id" integer NOT NULL,
  "db_id" integer(20) NOT NULL,
  "db" text(125) NOT NULL,
  "name" text(60),
  "sql" text,
  "type" integer(8) NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(32),
  "create_time"  datetime NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier_id" integer(20),
  "modifier" text(255),
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_db_sql_exec
CREATE TABLE IF NOT EXISTS "t_db_sql_exec" (
  "id" integer NOT NULL,
  "db_id" integer(20) NOT NULL,
  "db" text(128) NOT NULL,
  "table" text(128) NOT NULL,
  "type" text(255) NOT NULL,
  "sql" text(5000) NOT NULL,
  "old_value" text(5000),
  "remark" text(128),
  "create_time"  datetime NOT NULL,
  "creator" text(36) NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier" text(36) NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_machine
CREATE TABLE IF NOT EXISTS "t_machine" (
  "id" integer NOT NULL,
  "code" text(32),
  "name" text(32),
  "ip" text(50) NOT NULL,
  "port" integer(12) NOT NULL,
  "username" text(12) NOT NULL,
  "auth_method" integer(2),
  "password" text(100),
  "auth_cert_id" integer(20),
  "ssh_tunnel_machine_id" integer(20),
  "enable_recorder" integer(2),
  "status" integer(2) NOT NULL,
  "remark" text(255),
  "need_monitor" integer(2),
  "create_time"  datetime NOT NULL,
  "creator" text(16),
  "creator_id" integer(32),
  "update_time"  datetime NOT NULL,
  "modifier" text(12),
  "modifier_id" integer(32),
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_machine_cron_job
CREATE TABLE IF NOT EXISTS "t_machine_cron_job" (
  "id" integer NOT NULL,
  "key" text(32) NOT NULL,
  "name" text(255) NOT NULL,
  "cron" text(255) NOT NULL,
  "script" text,
  "remark" text(255),
  "status" integer(4),
  "save_exec_res_type" integer(4),
  "last_exec_time"  datetime,
  "creator_id" integer(20),
  "creator" text(32),
  "modifier_id" integer(20),
  "modifier" text(255),
  "create_time"  datetime,
  "update_time"  datetime,
  "is_deleted" integer(4) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_machine_cron_job_exec
CREATE TABLE IF NOT EXISTS "t_machine_cron_job_exec" (
  "id" integer NOT NULL,
  "cron_job_id" integer(20),
  "machine_id" integer(20),
  "status" integer(4),
  "res" text(1000),
  "exec_time"  datetime,
  "is_deleted" integer(4) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_machine_cron_job_relate
CREATE TABLE IF NOT EXISTS "t_machine_cron_job_relate" (
  "id" integer NOT NULL,
  "cron_job_id" integer(20),
  "machine_id" integer(20),
  "creator_id" integer(20),
  "creator" text(32),
  "create_time"  datetime,
  "is_deleted" integer(4) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_machine_file
CREATE TABLE IF NOT EXISTS "t_machine_file" (
  "id" integer NOT NULL,
  "machine_id" integer(20) NOT NULL,
  "name" text(45) NOT NULL,
  "path" text(45) NOT NULL,
  "type" text(45) NOT NULL,
  "creator_id" integer(20),
  "creator" text(45),
  "modifier_id" integer(20),
  "modifier" text(45),
  "create_time"  datetime NOT NULL,
  "update_time"  datetime,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_machine_monitor
CREATE TABLE IF NOT EXISTS "t_machine_monitor" (
  "id" integer NOT NULL,
  "machine_id" integer(20) NOT NULL,
  "cpu_rate" real(255,2),
  "mem_rate" real(255,2),
  "sys_load" text(32),
  "create_time"  datetime NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_machine_script
CREATE TABLE IF NOT EXISTS "t_machine_script" (
  "id" integer NOT NULL,
  "name" text(255) NOT NULL,
  "machine_id" integer(64) NOT NULL,
  "script" text,
  "params" text(512),
  "description" text(255),
  "type" integer(8),
  "creator_id" integer(20),
  "creator" text(32),
  "modifier_id" integer(20),
  "modifier" text(255),
  "create_time"  datetime,
  "update_time"  datetime,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_machine_script (id, name, machine_id, script, params, description, type, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1, 'sys_info', 9999999, '# 获取系统cpu信息
function get_cpu_info() {
  Physical_CPUs=$(grep "physical id" /proc/cpuinfo | sort | uniq | wc -l)
  Virt_CPUs=$(grep "processor" /proc/cpuinfo | wc -l)
  CPU_Kernels=$(grep "cores" /proc/cpuinfo | uniq | awk -F '': '' ''{print $2}'')
  CPU_Type=$(grep "model name" /proc/cpuinfo | awk -F '': '' ''{print $2}'' | sort | uniq)
  CPU_Arch=$(uname -m)
  echo -e ''\n-------------------------- CPU信息 --------------------------''
  cat <<EOF | column -t
物理CPU个数: $Physical_CPUs
逻辑CPU个数: $Virt_CPUs
每CPU核心数: $CPU_Kernels
CPU型号: $CPU_Type
CPU架构: $CPU_Arch
EOF
}

# 获取系统信息
function get_systatus_info() {
  sys_os=$(uname -o)
  sys_release=$(cat /etc/redhat-release)
  sys_kernel=$(uname -r)
  sys_hostname=$(hostname)
  sys_selinux=$(getenforce)
  sys_lang=$(echo $LANG)
  sys_lastreboot=$(who -b | awk ''{print $3,$4}'')
  echo -e ''-------------------------- 系统信息 --------------------------''
  cat <<EOF | column -t
系统: ${sys_os}
发行版本:   ${sys_release}
系统内核:   ${sys_kernel}
主机名:    ${sys_hostname}
selinux状态:  ${sys_selinux}
系统语言:   ${sys_lang}
系统最后重启时间:   ${sys_lastreboot}
EOF
}

get_systatus_info
#echo -e "\n"
get_cpu_info', NULL, '获取系统信息', 1, NULL, NULL, NULL, NULL, NULL, NULL, 0, NULL);
INSERT INTO t_machine_script (id, name, machine_id, script, params, description, type, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (2, 'get_process_by_name', 9999999, '#! /bin/bash
# Function: 根据输入的程序的名字过滤出所对应的PID，并显示出详细信息，如果有几个PID，则全部显示
NAME={{.processName}}
N=`ps -aux | grep $NAME | grep -v grep | wc -l`    ##统计进程总数
if [ $N -le 0 ];then
  echo "无该进程！"
fi
i=1
while [ $N -gt 0 ]
do
  echo "进程PID: `ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $2}''`"
  echo "进程命令：`ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $11}''`"
  echo "进程所属用户: `ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $1}''`"
  echo "CPU占用率：`ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $3}''`%"
  echo "内存占用率：`ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $4}''`%"
  echo "进程开始运行的时刻：`ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $9}''`"
  echo "进程运行的时间：`  ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $11}''`"
  echo "进程状态：`ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $8}''`"
  echo "进程虚拟内存：`ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $5}''`"
  echo "进程共享内存：`ps -aux | grep $NAME | grep -v grep | awk ''NR==''$i''{print $0}''| awk ''{print $6}''`"
  echo "***************************************************************"
  let N-- i++
done', '[{"name": "进程名","model": "processName", "placeholder": "请输入进程名"}]', '获取进程运行状态', 1, NULL, NULL, 1, 'admin', NULL, '2021-07-12 15:33:41', 0, NULL);
INSERT INTO t_machine_script (id, name, machine_id, script, params, description, type, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (3, 'sys_run_info', 9999999, '#!/bin/bash
# 获取要监控的本地服务器IP地址
IP=`ifconfig | grep inet | grep -vE ''inet6|127.0.0.1'' | awk ''{print $2}''`
echo "IP地址："$IP
 
# 获取cpu总核数
cpu_num=`grep -c "model name" /proc/cpuinfo`
echo "cpu总核数："$cpu_num
 
# 1、获取CPU利用率
################################################
#us 用户空间占用CPU百分比
#sy 内核空间占用CPU百分比
#ni 用户进程空间内改变过优先级的进程占用CPU百分比
#id 空闲CPU百分比
#wa 等待输入输出的CPU时间百分比
#hi 硬件中断
#si 软件中断
#################################################
# 获取用户空间占用CPU百分比
cpu_user=`top -b -n 1 | grep Cpu | awk ''{print $2}'' | cut -f 1 -d "%"`
echo "用户空间占用CPU百分比："$cpu_user
 
# 获取内核空间占用CPU百分比
cpu_system=`top -b -n 1 | grep Cpu | awk ''{print $4}'' | cut -f 1 -d "%"`
echo "内核空间占用CPU百分比："$cpu_system
 
# 获取空闲CPU百分比
cpu_idle=`top -b -n 1 | grep Cpu | awk ''{print $8}'' | cut -f 1 -d "%"`
echo "空闲CPU百分比："$cpu_idle
 
# 获取等待输入输出占CPU百分比
cpu_iowait=`top -b -n 1 | grep Cpu | awk ''{print $10}'' | cut -f 1 -d "%"`
echo "等待输入输出占CPU百分比："$cpu_iowait
 
#2、获取CPU上下文切换和中断次数
# 获取CPU中断次数
cpu_interrupt=`vmstat -n 1 1 | sed -n 3p | awk ''{print $11}''`
echo "CPU中断次数："$cpu_interrupt
 
# 获取CPU上下文切换次数
cpu_context_switch=`vmstat -n 1 1 | sed -n 3p | awk ''{print $12}''`
echo "CPU上下文切换次数："$cpu_context_switch
 
#3、获取CPU负载信息
# 获取CPU15分钟前到现在的负载平均值
cpu_load_15min=`uptime | awk ''{print $11}'' | cut -f 1 -d '',''`
echo "CPU 15分钟前到现在的负载平均值："$cpu_load_15min
 
# 获取CPU5分钟前到现在的负载平均值
cpu_load_5min=`uptime | awk ''{print $10}'' | cut -f 1 -d '',''`
echo "CPU 5分钟前到现在的负载平均值："$cpu_load_5min
 
# 获取CPU1分钟前到现在的负载平均值
cpu_load_1min=`uptime | awk ''{print $9}'' | cut -f 1 -d '',''`
echo "CPU 1分钟前到现在的负载平均值："$cpu_load_1min
 
# 获取任务队列(就绪状态等待的进程数)
cpu_task_length=`vmstat -n 1 1 | sed -n 3p | awk ''{print $1}''`
echo "CPU任务队列长度："$cpu_task_length
 
#4、获取内存信息
# 获取物理内存总量
mem_total=`free -h | grep Mem | awk ''{print $2}''`
echo "物理内存总量："$mem_total
 
# 获取操作系统已使用内存总量
mem_sys_used=`free -h | grep Mem | awk ''{print $3}''`
echo "已使用内存总量(操作系统)："$mem_sys_used
 
# 获取操作系统未使用内存总量
mem_sys_free=`free -h | grep Mem | awk ''{print $4}''`
echo "剩余内存总量(操作系统)："$mem_sys_free
 
# 获取应用程序已使用的内存总量
mem_user_used=`free | sed -n 3p | awk ''{print $3}''`
echo "已使用内存总量(应用程序)："$mem_user_used
 
# 获取应用程序未使用内存总量
mem_user_free=`free | sed -n 3p | awk ''{print $4}''`
echo "剩余内存总量(应用程序)："$mem_user_free
 
# 获取交换分区总大小
mem_swap_total=`free | grep Swap | awk ''{print $2}''`
echo "交换分区总大小："$mem_swap_total
 
# 获取已使用交换分区大小
mem_swap_used=`free | grep Swap | awk ''{print $3}''`
echo "已使用交换分区大小："$mem_swap_used
 
# 获取剩余交换分区大小
mem_swap_free=`free | grep Swap | awk ''{print $4}''`
echo "剩余交换分区大小："$mem_swap_free', NULL, '获取cpu、内存等系统运行状态', 1, NULL, NULL, NULL, NULL, NULL, '2021-04-25 15:07:16', 0, NULL);
INSERT INTO t_machine_script (id, name, machine_id, script, params, description, type, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (4, 'top', 9999999, 'top', NULL, '实时获取系统运行状态', 3, NULL, NULL, 1, 'admin', NULL, '2021-05-24 15:58:20', 0, NULL);
INSERT INTO t_machine_script (id, name, machine_id, script, params, description, type, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (18, 'disk-mem', 9999999, 'df -h', '', '磁盘空间查看', 1, 1, 'admin', 1, 'admin', '2021-07-16 10:49:53', '2021-07-16 10:49:53', 0, NULL);

-- Table: t_machine_term_op
CREATE TABLE IF NOT EXISTS "t_machine_term_op" (
  "id" integer NOT NULL,
  "machine_id" integer(20) NOT NULL,
  "username" text(60),
  "record_file_path" text(191),
  "creator_id" integer(20),
  "creator" text(191),
  "create_time"  datetime NOT NULL,
  "end_time"  datetime,
  "is_deleted" integer(4),
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_mongo
CREATE TABLE IF NOT EXISTS "t_mongo" (
  "id" integer NOT NULL,
  "code" text(32),
  "name" text(36) NOT NULL,
  "uri" text(255) NOT NULL,
  "ssh_tunnel_machine_id" integer(20),
  "create_time"  datetime NOT NULL,
  "creator_id" integer(20),
  "creator" text(36),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(36),
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_oauth2_account
CREATE TABLE IF NOT EXISTS "t_oauth2_account" (
  "id" integer NOT NULL,
  "account_id" integer(20) NOT NULL,
  "identity" text(64),
  "create_time"  datetime NOT NULL,
  "update_time"  datetime NOT NULL,
  "is_deleted" integer(4),
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_redis
CREATE TABLE IF NOT EXISTS "t_redis" (
  "id" integer NOT NULL,
  "code" text(32),
  "name" text(255),
  "host" text(255) NOT NULL,
  "username" text(32),
  "password" text(100),
  "db" text(64),
  "mode" text(32),
  "ssh_tunnel_machine_id" integer(20),
  "remark" text(125),
  "creator" text(32),
  "creator_id" integer(32),
  "create_time"  datetime,
  "modifier" text(32),
  "modifier_id" integer(20),
  "update_time"  datetime,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_sys_account
CREATE TABLE IF NOT EXISTS "t_sys_account" (
  "id" integer NOT NULL,
  "name" text(30) NOT NULL,
  "username" text(30) NOT NULL,
  "password" text(64) NOT NULL,
  "status" integer(4),
  "otp_secret" text(100),
  "last_login_time"  datetime,
  "last_login_ip" text(50),
  "create_time"  datetime NOT NULL,
  "creator_id" integer(255) NOT NULL,
  "creator" text(12) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier_id" integer(255) NOT NULL,
  "modifier" text(12) NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_sys_account (id, name, username, password, status, otp_secret, last_login_time, last_login_ip, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (1, '管理员', 'admin', '$2a$10$w3Wky2U.tinvR7c/s0aKPuwZsIu6pM1/DMJalwBDMbE6niHIxVrrm', 1, '', '2022-10-26 20:03:48', '::1', '2020-01-01 19:00:00', 1, 'admin', '2020-01-01 19:00:00', 1, 'admin', 0, NULL);

-- Table: t_sys_account_role
CREATE TABLE IF NOT EXISTS "t_sys_account_role" (
  "id" integer NOT NULL,
  "account_id" integer(20) NOT NULL,
  "role_id" integer(20) NOT NULL,
  "creator" text(45),
  "creator_id" integer(20),
  "create_time"  datetime NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_sys_config
CREATE TABLE IF NOT EXISTS "t_sys_config" (
  "id" integer NOT NULL,
  "name" text(60) NOT NULL,
  "key" text(120) NOT NULL,
  "params" text(1500),
  "value" text(1500),
  "remark" text(255),
  "permission" text(255),
  "create_time"  datetime NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(36) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "modifier" text(36) NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (3, '账号登录安全设置', 'AccountLoginSecurity', '[{"name":"登录验证码","model":"useCaptcha","placeholder":"是否启用登录验证码","options":"true,false"},{"name":"双因素校验(OTP)","model":"useOtp","placeholder":"是否启用双因素(OTP)校验","options":"true,false"},{"name":"OTP签发人","model":"otpIssuer","placeholder":"otp签发人"},{"name":"允许失败次数","model":"loginFailCount","placeholder":"登录失败n次后禁止登录"},{"name":"禁止登录时间","model":"loginFailMin","placeholder":"登录失败指定次数后禁止m分钟内再次登录"}]', '{"useCaptcha":"true","useOtp":"false","loginFailCount":"5","loginFailMin":"10","otpIssuer":"mayfly-go"}', '系统账号登录相关安全设置', 'all', '2023-06-17 11:02:11', 1, 'admin', '2023-06-17 14:18:07', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (4, 'oauth2登录配置', 'Oauth2Login', '[{"name":"是否启用","model":"enable","placeholder":"是否启用oauth2登录","options":"true,false"},{"name":"名称","model":"name","placeholder":"oauth2名称"},{"name":"Client ID","model":"clientId","placeholder":"Client ID"},{"name":"Client Secret","model":"clientSecret","placeholder":"Client Secret"},{"name":"Authorization URL","model":"authorizationURL","placeholder":"Authorization URL"},{"name":"AccessToken URL","model":"accessTokenURL","placeholder":"AccessToken URL"},{"name":"Redirect URL","model":"redirectURL","placeholder":"本系统地址"},{"name":"Scopes","model":"scopes","placeholder":"Scopes"},{"name":"Resource URL","model":"resourceURL","placeholder":"获取用户信息资源地址"},{"name":"UserIdentifier","model":"userIdentifier","placeholder":"用户唯一标识字段;格式为type:fieldPath(string:username)"},{"name":"是否自动注册","model":"autoRegister","placeholder":"","options":"true,false"}]', '', 'oauth2登录相关配置信息', 'admin,', '2023-07-22 13:58:51', 1, 'admin', '2023-07-22 19:34:37', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (5, 'ldap登录配置', 'LdapLogin', '[{"name":"是否启用","model":"enable","placeholder":"是否启用","options":"true,false"},{"name":"host","model":"host","placeholder":"host"},{"name":"port","model":"port","placeholder":"port"},{"name":"bindDN","model":"bindDN","placeholder":"LDAP 服务的管理员账号，如: \"cn=admin,dc=example,dc=com\""},{"name":"bindPwd","model":"bindPwd","placeholder":"LDAP 服务的管理员密码"},{"name":"baseDN","model":"baseDN","placeholder":"用户所在的 base DN, 如: \"ou=users,dc=example,dc=com\""},{"name":"userFilter","model":"userFilter","placeholder":"过滤用户的方式, 如: \"(uid=%s)、(&(objectClass=organizationalPerson)(uid=%s))\""},{"name":"uidMap","model":"uidMap","placeholder":"用户id和 LDAP 字段名之间的映射关系,如: cn"},{"name":"udnMap","model":"udnMap","placeholder":"用户姓名(dispalyName)和 LDAP 字段名之间的映射关系,如: displayName"},{"name":"emailMap","model":"emailMap","placeholder":"用户email和 LDAP 字段名之间的映射关系"},{"name":"skipTLSVerify","model":"skipTLSVerify","placeholder":"客户端是否跳过 TLS 证书验证","options":"true,false"},{"name":"安全协议","model":"securityProtocol","placeholder":"安全协议（为Null不使用安全协议），如: StartTLS, LDAPS","options":"Null,StartTLS,LDAPS"}]', '', 'ldap登录相关配置', 'admin,', '2023-08-25 21:47:20', 1, 'admin', '2023-08-25 22:56:07', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (6, '系统全局样式设置', 'SysStyleConfig', '[{"model":"logoIcon","name":"logo图标","placeholder":"系统logo图标（base64编码, 建议svg格式，不超过10k）","required":false},{"model":"title","name":"菜单栏标题","placeholder":"系统菜单栏标题展示","required":false},{"model":"viceTitle","name":"登录页标题","placeholder":"登录页标题展示","required":false},{"model":"useWatermark","name":"是否启用水印","placeholder":"是否启用系统水印","options":"true,false","required":false},{"model":"watermarkContent","name":"水印补充信息","placeholder":"额外水印信息","required":false}]', '{"title":"mayfly-go","viceTitle":"mayfly-go","logoIcon":"","useWatermark":"true","watermarkContent":""}', '系统icon、标题、水印信息等配置', 'all', '2024-01-04 15:17:18', 1, 'admin', '2024-01-05 09:40:44', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (7, '数据库查询最大结果集', 'DbQueryMaxCount', '[]', '200', '允许sql查询的最大结果集数。注: 0=不限制', 'all', '2023-02-11 14:29:03', 1, 'admin', '2023-02-11 14:40:56', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (8, '数据库是否记录查询SQL', 'DbSaveQuerySQL', '[]', '0', '1: 记录、0:不记录', 'all', '2023-02-11 16:07:14', 1, 'admin', '2023-02-11 16:44:17', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (9, '机器相关配置', 'MachineConfig', '[{"name":"终端回放存储路径","model":"terminalRecPath","placeholder":"终端回放存储路径"},{"name":"uploadMaxFileSize","model":"uploadMaxFileSize","placeholder":"允许上传的最大文件大小(1MB\\2GB等)"}]', '{"terminalRecPath":"./rec","uploadMaxFileSize":"1GB"}', '机器相关配置，如终端回放路径等', 'admin,', '2023-07-13 16:26:44', 1, 'admin', '2023-11-09 22:01:31', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (10, '数据库备份恢复', 'DbBackupRestore', '[{"model":"backupPath","name":"备份路径","placeholder":"备份文件存储路径"}]', '{"backupPath":"./db/backup"}', '', 'admin,', '2023-12-29 09:55:26', 1, 'admin', '2023-12-29 15:45:24', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (11, 'Mysql可执行文件', 'MysqlBin', '[{"model":"path","name":"路径","placeholder":"可执行文件路径","required":true},{"model":"mysql","name":"mysql","placeholder":"mysql命令路径(空则为 路径/mysql)","required":false},{"model":"mysqldump","name":"mysqldump","placeholder":"mysqldump命令路径(空则为 路径/mysqldump)","required":false},{"model":"mysqlbinlog","name":"mysqlbinlog","placeholder":"mysqlbinlog命令路径(空则为 路径/mysqlbinlog)","required":false}]', '{"mysql":"","mysqldump":"","mysqlbinlog":"","path":"./db/mysql/bin"}', '', 'admin,', '2023-12-29 10:01:33', 1, 'admin', '2023-12-29 13:34:40', 1, 'admin', 0, NULL);
INSERT INTO t_sys_config (id, name, key, params, value, remark, permission, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (12, 'MariaDB可执行文件', 'MariadbBin', '[{"model":"path","name":"路径","placeholder":"可执行文件路径","required":true},{"model":"mysql","name":"mysql","placeholder":"mysql命令路径(空则为 路径/mysql)","required":false},{"model":"mysqldump","name":"mysqldump","placeholder":"mysqldump命令路径(空则为 路径/mysqldump)","required":false},{"model":"mysqlbinlog","name":"mysqlbinlog","placeholder":"mysqlbinlog命令路径(空则为 路径/mysqlbinlog)","required":false}]', '{"mysql":"","mysqldump":"","mysqlbinlog":"","path":"./db/mariadb/bin"}', '', 'admin,', '2023-12-29 10:01:33', 1, 'admin', '2023-12-29 13:34:40', 1, 'admin', 0, NULL);

-- Table: t_sys_log
CREATE TABLE IF NOT EXISTS "t_sys_log" (
  "id" integer NOT NULL,
  "type" integer(4) NOT NULL,
  "description" text(255),
  "req_param" text(2000),
  "resp" text(1000),
  "creator" text(36) NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "create_time"  datetime NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_sys_msg
CREATE TABLE IF NOT EXISTS "t_sys_msg" (
  "id" integer NOT NULL,
  "type" integer(255),
  "msg" text(2000) NOT NULL,
  "recipient_id" integer(20),
  "creator_id" integer(20),
  "creator" text(36),
  "create_time"  datetime NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_sys_resource
CREATE TABLE IF NOT EXISTS "t_sys_resource" (
  "id" integer NOT NULL,
  "pid" integer(11) NOT NULL,
  "ui_path" text(200),
  "type" integer(4) NOT NULL,
  "status" integer(11) NOT NULL,
  "name" text(255) NOT NULL,
  "code" text(255),
  "weight" integer(11),
  "meta" text(455),
  "creator_id" integer(20) NOT NULL,
  "creator" text(255) NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "modifier" text(255) NOT NULL,
  "create_time"  datetime NOT NULL,
  "update_time"  datetime NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1, 0, 'Aexqq77l/', 1, 1, '首页', '/home', 10000000, '{"component":"home/Home","icon":"HomeFilled","isAffix":true,"isKeepAlive":true,"routeName":"Home"}', 1, 'admin', 1, 'admin', '2021-05-25 16:44:41', '2023-03-14 14:27:07', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (2, 0, '12sSjal1/', 1, 1, '机器管理', '/machine', 49999998, '{"icon":"Monitor","isKeepAlive":true,"redirect":"machine/list","routeName":"Machine"}', 1, 'admin', 1, 'admin', '2021-05-25 16:48:16', '2022-10-06 14:58:49', 0, NULL);
INSERT INTO t_sys_resource (id, pid, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, ui_path, is_deleted, delete_time) VALUES(1707206386, 2, 1, 1, '机器操作', 'machines-op', 1, '{"component":"ops/machine/MachineOp","icon":"Monitor","isKeepAlive":true,"routeName":"MachineOp"}', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-06 15:59:46', '2024-02-06 16:24:21', 'PDPt6217/', 0, NULL);
INSERT INTO t_sys_resource (id, pid, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, ui_path, is_deleted, delete_time) VALUES(1707206421, 1707206386, 2, 1, '基本权限', 'machine-op', 1707206421, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-06 16:00:22', '2024-02-06 16:00:22', 'PDPt6217/kQXTYvuM/', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (3, 2, '12sSjal1/lskeiql1/', 1, 1, '机器列表', 'machines', 20000000, '{"component":"ops/machine/MachineList","icon":"Monitor","isKeepAlive":true,"routeName":"MachineList"}', 2, 'admin', 1, 'admin', '2021-05-25 16:50:04', '2023-03-15 17:14:44', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (4, 0, 'Xlqig32x/', 1, 1, '系统管理', '/sys', 60000001, '{"icon":"Setting","isKeepAlive":true,"redirect":"/sys/resources","routeName":"sys"}', 1, 'admin', 1, 'admin', '2021-05-26 15:20:20', '2022-10-06 14:59:53', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (5, 4, 'Xlqig32x/UGxla231/', 1, 1, '资源管理', 'resources', 9999999, '{"component":"system/resource/ResourceList","icon":"Menu","isKeepAlive":true,"routeName":"ResourceList"}', 1, 'admin', 1, 'admin', '2021-05-26 15:23:07', '2023-03-14 15:44:34', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (11, 4, 'Xlqig32x/lxqSiae1/', 1, 1, '角色管理', 'roles', 10000001, '{"component":"system/role/RoleList","icon":"Menu","isKeepAlive":true,"routeName":"RoleList"}', 1, 'admin', 1, 'admin', '2021-05-27 11:15:35', '2023-03-14 15:44:22', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (12, 3, '12sSjal1/lskeiql1/Alw1Xkq3/', 2, 1, '机器终端按钮', 'machine:terminal', 40000000, '', 1, 'admin', 1, 'admin', '2021-05-28 14:06:02', '2021-05-31 17:47:59', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (14, 4, 'Xlqig32x/sfslfel/', 1, 1, '账号管理', 'accounts', 9999999, '{"component":"system/account/AccountList","icon":"Menu","isKeepAlive":true,"routeName":"AccountList"}', 1, 'admin', 1, 'admin', '2021-05-28 14:56:25', '2023-03-14 15:44:10', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (15, 3, '12sSjal1/lskeiql1/Lsew24Kx/', 2, 1, '文件管理按钮', 'machine:file', 50000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:44:37', '2021-05-31 17:48:07', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (16, 3, '12sSjal1/lskeiql1/exIsqL31/', 2, 1, '机器添加按钮', 'machine:add', 10000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:46:11', '2021-05-31 19:34:15', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (17, 3, '12sSjal1/lskeiql1/Liwakg2x/', 2, 1, '机器编辑按钮', 'machine:update', 20000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:46:23', '2021-05-31 19:34:18', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (18, 3, '12sSjal1/lskeiql1/Lieakenx/', 2, 1, '机器删除按钮', 'machine:del', 30000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:46:36', '2021-05-31 19:34:17', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (19, 14, 'Xlqig32x/sfslfel/UUiex2xA/', 2, 1, '角色分配按钮', 'account:saveRoles', 10000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:50:51', '2021-05-31 19:19:30', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (20, 11, 'Xlqig32x/lxqSiae1/EMq2Kxq3/', 2, 1, '分配菜单&权限按钮', 'role:saveResources', 10000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 17:51:41', '2021-05-31 19:33:37', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (21, 14, 'Xlqig32x/sfslfel/Uexax2xA/', 2, 1, '账号删除按钮', 'account:del', 20000000, 'null', 1, 'admin', 1, 'admin', '2021-05-31 18:02:01', '2021-06-10 17:12:17', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (22, 11, 'Xlqig32x/lxqSiae1/Elxq2Kxq3/', 2, 1, '角色删除按钮', 'role:del', 20000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:02:29', '2021-05-31 19:33:38', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (23, 11, 'Xlqig32x/lxqSiae1/342xKxq3/', 2, 1, '角色新增按钮', 'role:add', 30000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:02:44', '2021-05-31 19:33:39', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (24, 11, 'Xlqig32x/lxqSiae1/LexqKxq3/', 2, 1, '角色编辑按钮', 'role:update', 40000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:02:57', '2021-05-31 19:33:40', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (25, 5, 'Xlqig32x/UGxla231/Elxq23XK/', 2, 1, '资源新增按钮', 'resource:add', 10000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:03:33', '2021-05-31 19:31:47', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (26, 5, 'Xlqig32x/UGxla231/eloq23XK/', 2, 1, '资源删除按钮', 'resource:delete', 20000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:03:47', '2021-05-31 19:29:40', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (27, 5, 'Xlqig32x/UGxla231/JExq23XK/', 2, 1, '资源编辑按钮', 'resource:update', 30000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:04:03', '2021-05-31 19:29:40', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (28, 5, 'Xlqig32x/UGxla231/Elex13XK/', 2, 1, '资源禁用启用按钮', 'resource:changeStatus', 40000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 18:04:33', '2021-05-31 18:04:33', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (29, 14, 'Xlqig32x/sfslfel/xlawx2xA/', 2, 1, '账号添加按钮', 'account:add', 30000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 19:23:42', '2021-05-31 19:23:42', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (30, 14, 'Xlqig32x/sfslfel/32xax2xA/', 2, 1, '账号编辑修改按钮', 'account:update', 40000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 19:23:58', '2021-05-31 19:23:58', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (31, 14, 'Xlqig32x/sfslfel/eubale13/', 2, 1, '账号管理基本权限', 'account', 10000000, 'null', 1, 'admin', 1, 'admin', '2021-05-31 21:25:06', '2021-06-22 11:20:34', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (32, 5, 'Xlqig32x/UGxla231/321q23XK/', 2, 1, '资源管理基本权限', 'resource', 10000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 21:25:25', '2021-05-31 21:25:25', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (33, 11, 'Xlqig32x/lxqSiae1/908xKxq3/', 2, 1, '角色管理基本权限', 'role', 10000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 21:25:40', '2021-05-31 21:25:40', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (34, 14, 'Xlqig32x/sfslfel/32alx2xA/', 2, 1, '账号启用禁用按钮', 'account:changeStatus', 50000000, NULL, 1, 'admin', 1, 'admin', '2021-05-31 21:29:48', '2021-05-31 21:29:48', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (36, 0, 'dbms23ax/', 1, 1, 'DBMS', '/dbms', 49999999, '{"icon":"Coin","isKeepAlive":true,"routeName":"DBMS"}', 1, 'admin', 1, 'admin', '2021-06-01 14:01:33', '2023-03-15 17:31:08', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (37, 3, '12sSjal1/lskeiql1/Keiqkx4L/', 2, 1, '添加文件配置', 'machine:addFile', 60000000, 'null', 1, 'admin', 1, 'admin', '2021-06-01 19:54:23', '2021-06-01 19:54:23', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (38, 36, 'dbms23ax/exaeca2x/', 1, 1, '数据操作', 'sql-exec', 10000000, '{"component":"ops/db/SqlExec","icon":"Coin","isKeepAlive":true,"routeName":"SqlExec"}', 1, 'admin', 1, 'admin', '2021-06-03 09:09:29', '2023-03-15 17:31:21', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (39, 0, 'sl3as23x/', 1, 1, '个人中心', '/personal', 19999999, '{"component":"personal/index","icon":"UserFilled","isHide":true,"isKeepAlive":true,"routeName":"Personal"}', 1, 'admin', 1, 'admin', '2021-06-03 14:25:35', '2023-03-14 14:28:36', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (40, 3, '12sSjal1/lskeiql1/Keal2Xke/', 2, 1, '文件管理-新增按钮', 'machine:file:add', 70000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:06:26', '2021-06-08 11:12:28', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (41, 3, '12sSjal1/lskeiql1/Ihfs2xaw/', 2, 1, '文件管理-删除按钮', 'machine:file:del', 80000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:06:49', '2021-06-08 11:06:49', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (42, 3, '12sSjal1/lskeiql1/3ldkxJDx/', 2, 1, '文件管理-写入or下载文件权限', 'machine:file:write', 90000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:07:27', '2021-06-08 11:07:27', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (43, 3, '12sSjal1/lskeiql1/Ljewix43/', 2, 1, '文件管理-文件上传按钮', 'machine:file:upload', 100000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:07:42', '2021-06-08 11:07:42', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (44, 3, '12sSjal1/lskeiql1/L12wix43/', 2, 1, '文件管理-删除文件按钮', 'machine:file:rm', 110000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:08:12', '2021-06-08 11:08:12', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (45, 3, '12sSjal1/lskeiql1/Ljewisd3/', 2, 1, '脚本管理-保存脚本按钮', 'machine:script:save', 120000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:09:01', '2021-06-08 11:09:01', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (46, 3, '12sSjal1/lskeiql1/Ljeew43/', 2, 1, '脚本管理-删除按钮', 'machine:script:del', 130000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:09:27', '2021-06-08 11:09:27', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (47, 3, '12sSjal1/lskeiql1/ODewix43/', 2, 1, '脚本管理-执行按钮', 'machine:script:run', 140000000, 'null', 1, 'admin', 1, 'admin', '2021-06-08 11:09:50', '2021-06-08 11:09:50', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (49, 36, 'dbms23ax/xleaiec2/', 1, 1, '数据库管理', 'dbs', 20000000, '{"component":"ops/db/DbList","icon":"Coin","isKeepAlive":true,"routeName":"DbList"}', 1, 'admin', 1, 'admin', '2021-07-07 15:13:55', '2023-03-15 17:31:28', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (54, 49, 'dbms23ax/xleaiec2/leix3Axl/', 2, 1, '数据库保存', 'db:save', 10000000, 'null', 1, 'admin', 1, 'admin', '2021-07-08 17:30:36', '2021-07-08 17:31:05', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (55, 49, 'dbms23ax/xleaiec2/ygjL3sxA/', 2, 1, '数据库删除', 'db:del', 20000000, 'null', 1, 'admin', 1, 'admin', '2021-07-08 17:30:48', '2021-07-08 17:30:48', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (57, 3, '12sSjal1/lskeiql1/OJewex43/', 2, 1, '基本权限', 'machine', 10000000, 'null', 1, 'admin', 1, 'admin', '2021-07-09 10:48:02', '2021-07-09 10:48:02', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (58, 49, 'dbms23ax/xleaiec2/AceXe321/', 2, 1, '基本权限', 'db', 10000000, 'null', 1, 'admin', 1, 'admin', '2021-07-09 10:48:22', '2021-07-09 10:48:22', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (59, 38, 'dbms23ax/exaeca2x/ealcia23/', 2, 1, '基本权限', 'db:exec', 10000000, 'null', 1, 'admin', 1, 'admin', '2021-07-09 10:50:13', '2021-07-09 10:50:13', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (60, 0, 'RedisXq4/', 1, 1, 'Redis', '/redis', 50000001, '{"icon":"iconfont icon-redis","isKeepAlive":true,"routeName":"RDS"}', 1, 'admin', 1, 'admin', '2021-07-19 20:15:41', '2023-03-15 16:44:59', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (61, 60, 'RedisXq4/Exitx4al/', 1, 1, '数据操作', 'data-operation', 10000000, '{"component":"ops/redis/DataOperation","icon":"iconfont icon-redis","isKeepAlive":true,"routeName":"DataOperation"}', 1, 'admin', 1, 'admin', '2021-07-19 20:17:29', '2023-03-15 16:37:50', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (62, 61, 'RedisXq4/Exitx4al/LSjie321/', 2, 1, '基本权限', 'redis:data', 10000000, 'null', 1, 'admin', 1, 'admin', '2021-07-19 20:18:54', '2021-07-19 20:18:54', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (63, 60, 'RedisXq4/Eoaljc12/', 1, 1, 'redis管理', 'manage', 20000000, '{"component":"ops/redis/RedisList","icon":"iconfont icon-redis","isKeepAlive":true,"routeName":"RedisList"}', 1, 'admin', 1, 'admin', '2021-07-20 10:48:04', '2023-03-15 16:38:00', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (64, 63, 'RedisXq4/Eoaljc12/IoxqAd31/', 2, 1, '基本权限', 'redis:manage', 10000000, 'null', 1, 'admin', 1, 'admin', '2021-07-20 10:48:26', '2021-07-20 10:48:26', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (71, 61, 'RedisXq4/Exitx4al/IUlxia23/', 2, 1, '数据保存', 'redis:data:save', 60000000, 'null', 1, 'admin', 1, 'admin', '2021-08-17 11:20:37', '2021-08-17 11:20:37', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (72, 3, '12sSjal1/lskeiql1/LIEwix43/', 2, 1, '终止进程', 'machine:killprocess', 60000000, 'null', 1, 'admin', 1, 'admin', '2021-08-17 11:20:37', '2021-08-17 11:20:37', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (79, 0, 'Mongo452/', 1, 1, 'Mongo', '/mongo', 50000002, '{"icon":"iconfont icon-mongo","isKeepAlive":true,"routeName":"Mongo"}', 1, 'admin', 1, 'admin', '2022-05-13 14:00:41', '2023-03-16 14:23:22', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (80, 79, 'Mongo452/eggago31/', 1, 1, '数据操作', 'mongo-data-operation', 10000000, '{"component":"ops/mongo/MongoDataOp","icon":"iconfont icon-mongo","isKeepAlive":true,"routeName":"MongoDataOp"}', 1, 'admin', 1, 'admin', '2022-05-13 14:03:58', '2023-03-15 17:15:02', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (81, 80, 'Mongo452/eggago31/egjglal3/', 2, 1, '基本权限', 'mongo:base', 10000000, 'null', 1, 'admin', 1, 'admin', '2022-05-13 14:04:16', '2022-05-13 14:04:16', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (82, 79, 'Mongo452/ghxagl43/', 1, 1, 'Mongo管理', 'mongo-manage', 20000000, '{"component":"ops/mongo/MongoList","icon":"iconfont icon-mongo","isKeepAlive":true,"routeName":"MongoList"}', 1, 'admin', 1, 'admin', '2022-05-16 18:13:06', '2023-03-15 17:26:55', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (83, 82, 'Mongo452/ghxagl43/egljbla3/', 2, 1, '基本权限', 'mongo:manage:base', 10000000, 'null', 1, 'admin', 1, 'admin', '2022-05-16 18:13:25', '2022-05-16 18:13:25', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (84, 4, 'Xlqig32x/exlaeAlx/', 1, 1, '操作日志', 'syslogs', 20000000, '{"component":"system/syslog/SyslogList","icon":"Tickets","routeName":"SyslogList"}', 1, 'admin', 1, 'admin', '2022-07-13 19:57:07', '2023-03-14 15:44:45', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (85, 84, 'Xlqig32x/exlaeAlx/3xlqeXql/', 2, 1, '操作日志基本权限', 'syslog', 10000000, 'null', 1, 'admin', 1, 'admin', '2022-07-13 19:57:55', '2022-07-13 19:57:55', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (87, 4, 'Xlqig32x/Ulxaee23/', 1, 1, '系统配置', 'configs', 10000002, '{"component":"system/config/ConfigList","icon":"Setting","isKeepAlive":true,"routeName":"ConfigList"}', 1, 'admin', 1, 'admin', '2022-08-25 22:18:55', '2023-03-15 11:06:07', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (88, 87, 'Xlqig32x/Ulxaee23/exlqguA3/', 2, 1, '基本权限', 'config:base', 10000000, 'null', 1, 'admin', 1, 'admin', '2022-08-25 22:19:35', '2022-08-25 22:19:35', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (93, 0, 'Tag3fhad/', 1, 1, '标签管理', '/tag', 20000001, '{"icon":"CollectionTag","isKeepAlive":true,"routeName":"Tag"}', 1, 'admin', 1, 'admin', '2022-10-24 15:18:40', '2022-10-24 15:24:29', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (94, 93, 'Tag3fhad/glxajg23/', 1, 1, '标签树', 'tag-trees', 10000000, '{"component":"ops/tag/TagTreeList","icon":"CollectionTag","isKeepAlive":true,"routeName":"TagTreeList"}', 1, 'admin', 1, 'admin', '2022-10-24 15:19:40', '2023-03-14 14:30:51', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (95, 93, 'Tag3fhad/Bjlag32x/', 1, 1, '团队管理', 'teams', 20000000, '{"component":"ops/tag/TeamList","icon":"UserFilled","isKeepAlive":true,"routeName":"TeamList"}', 1, 'admin', 1, 'admin', '2022-10-24 15:20:09', '2023-03-14 14:31:03', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (96, 94, 'Tag3fhad/glxajg23/gkxagt23/', 2, 1, '保存标签', 'tag:save', 10000000, 'null', 1, 'admin', 1, 'admin', '2022-10-24 15:20:40', '2022-10-26 13:58:36', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (97, 95, 'Tag3fhad/Bjlag32x/GJslag32/', 2, 1, '保存团队', 'team:save', 10000000, 'null', 1, 'admin', 1, 'admin', '2022-10-24 15:20:57', '2022-10-26 13:58:56', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (98, 94, 'Tag3fhad/glxajg23/xjgalte2/', 2, 1, '删除标签', 'tag:del', 20000000, 'null', 1, 'admin', 1, 'admin', '2022-10-26 13:58:47', '2022-10-26 13:58:47', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (99, 95, 'Tag3fhad/Bjlag32x/Gguca23x/', 2, 1, '删除团队', 'team:del', 20000000, 'null', 1, 'admin', 1, 'admin', '2022-10-26 13:59:06', '2022-10-26 13:59:06', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (100, 95, 'Tag3fhad/Bjlag32x/Lgidsq32/', 2, 1, '新增团队成员', 'team:member:save', 30000000, 'null', 1, 'admin', 1, 'admin', '2022-10-26 13:59:27', '2022-10-26 13:59:27', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (101, 95, 'Tag3fhad/Bjlag32x/Lixaue3G/', 2, 1, '移除团队成员', 'team:member:del', 40000000, 'null', 1, 'admin', 1, 'admin', '2022-10-26 13:59:43', '2022-10-26 13:59:43', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (102, 95, 'Tag3fhad/Bjlag32x/Oygsq3xg/', 2, 1, '保存团队标签', 'team:tag:save', 50000000, 'null', 1, 'admin', 1, 'admin', '2022-10-26 13:59:57', '2022-10-26 13:59:57', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (103, 2, '12sSjal1/exahgl32/', 1, 1, '授权凭证', 'authcerts', 60000000, '{"component":"ops/machine/authcert/AuthCertList","icon":"Unlock","isKeepAlive":true,"routeName":"AuthCertList"}', 1, 'admin', 1, 'admin', '2023-02-23 11:36:26', '2023-03-14 14:33:28', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (104, 103, '12sSjal1/exahgl32/egxahg24/', 2, 1, '基本权限', 'authcert', 10000000, 'null', 1, 'admin', 1, 'admin', '2023-02-23 11:37:24', '2023-02-23 11:37:24', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (105, 103, '12sSjal1/exahgl32/yglxahg2/', 2, 1, '保存权限', 'authcert:save', 20000000, 'null', 1, 'admin', 1, 'admin', '2023-02-23 11:37:54', '2023-02-23 11:37:54', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (106, 103, '12sSjal1/exahgl32/Glxag234/', 2, 1, '删除权限', 'authcert:del', 30000000, 'null', 1, 'admin', 1, 'admin', '2023-02-23 11:38:09', '2023-02-23 11:38:09', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (108, 61, 'RedisXq4/Exitx4al/Gxlagheg/', 2, 1, '数据删除', 'redis:data:del', 30000000, 'null', 1, 'admin', 1, 'admin', '2023-03-14 17:20:00', '2023-03-14 17:20:00', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (109, 3, '12sSjal1/lskeiql1/KMdsix43/', 2, 1, '关闭连接', 'machine:close-cli', 60000000, 'null', 1, 'admin', 1, 'admin', '2023-03-16 16:11:04', '2023-03-16 16:11:04', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (128, 87, 'Xlqig32x/Ulxaee23/MoOWr2N0/', 2, 1, '配置保存', 'config:save', 1687315135, 'null', 1, 'admin', 1, 'admin', '2023-06-21 10:38:55', '2023-06-21 10:38:55', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (130, 2, '12sSjal1/W9XKiabq/', 1, 1, '计划任务', '/machine/cron-job', 1689646396, '{"component":"ops/machine/cronjob/CronJobList","icon":"AlarmClock","isKeepAlive":true,"routeName":"CronJobList"}', 1, 'admin', 1, 'admin', '2023-07-18 10:13:16', '2023-07-18 10:14:06', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (131, 130, '12sSjal1/W9XKiabq/gEOqr2pD/', 2, 1, '保存计划任务', 'machine:cronjob:save', 1689860087, 'null', 1, 'admin', 1, 'admin', '2023-07-20 21:34:47', '2023-07-20 21:34:47', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (132, 130, '12sSjal1/W9XKiabq/zxXM23i0/', 2, 1, '删除计划任务', 'machine:cronjob:del', 1689860102, 'null', 1, 'admin', 1, 'admin', '2023-07-20 21:35:02', '2023-07-20 21:35:02', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (133, 80, 'Mongo452/eggago31/xvpKk36u/', 2, 1, '保存数据', 'mongo:data:save', 1692674943, 'null', 1, 'admin', 1, 'admin', '2023-08-22 11:29:04', '2023-08-22 11:29:11', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (134, 80, 'Mongo452/eggago31/3sblw1Wb/', 2, 1, '删除数据', 'mongo:data:del', 1692674964, 'null', 1, 'admin', 1, 'admin', '2023-08-22 11:29:24', '2023-08-22 11:29:24', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (135, 36, 'dbms23ax/X0f4BxT0/', 1, 1, '数据库实例', 'instances', 1693040706, '{"component":"ops/db/InstanceList","icon":"Coin","isKeepAlive":true,"routeName":"InstanceList"}', 1, 'admin', 1, 'admin', '2023-08-26 09:05:07', '2023-08-29 22:35:11', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (136, 135, 'dbms23ax/X0f4BxT0/D23fUiBr/', 2, 1, '实例保存', 'db:instance:save', 1693041001, 'null', 1, 'admin', 1, 'admin', '2023-08-26 09:10:02', '2023-08-26 09:10:02', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (137, 135, 'dbms23ax/X0f4BxT0/mJlBeTCs/', 2, 1, '基本权限', 'db:instance', 1693041055, 'null', 1, 'admin', 1, 'admin', '2023-08-26 09:10:55', '2023-08-26 09:10:55', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (138, 135, 'dbms23ax/X0f4BxT0/Sgg8uPwz/', 2, 1, '实例删除', 'db:instance:del', 1693041084, 'null', 1, 'admin', 1, 'admin', '2023-08-26 09:11:24', '2023-08-26 09:11:24', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (150, 36, 'Jra0n7De/', 1, 1, '数据同步', 'sync', 1693040707, '{"component":"ops/db/SyncTaskList","icon":"Coin","isKeepAlive":true,"routeName":"SyncTaskList"}', 12, 'liuzongyang', 12, 'liuzongyang', '2023-12-22 09:51:34', '2023-12-27 10:16:57', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (151, 150, 'Jra0n7De/uAnHZxEV/', 2, 1, '基本权限', 'db:sync', 1703641202, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2023-12-27 09:40:02', '2023-12-27 09:40:02', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (152, 150, 'Jra0n7De/zvAMo2vk/', 2, 1, '编辑', 'db:sync:save', 1703641320, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2023-12-27 09:42:00', '2023-12-27 09:42:12', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (153, 150, 'Jra0n7De/pLOA2UYz/', 2, 1, '删除', 'db:sync:del', 1703641342, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2023-12-27 09:42:22', '2023-12-27 09:42:22', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (154, 150, 'Jra0n7De/VBt68CDx/', 2, 1, '启停', 'db:sync:status', 1703641364, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2023-12-27 09:42:45', '2023-12-27 09:42:45', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (155, 150, 'Jra0n7De/PigmSGVg/', 2, 1, '日志', 'db:sync:log', 1704266866, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-01-03 15:27:47', '2024-01-03 15:27:47', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (160, 49, 'dbms23ax/xleaiec2/3NUXQFIO/', 2, 1, '数据库备份', 'db:backup', 1705973876, 'null', 1, 'admin', 1, 'admin', '2024-01-23 09:37:56', '2024-01-23 09:37:56', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (161, 49, 'dbms23ax/xleaiec2/ghErkTdb/', 2, 1, '数据库恢复', 'db:restore', 1705973909, 'null', 1, 'admin', 1, 'admin', '2024-01-23 09:38:29', '2024-01-23 09:38:29', 0, NULL);

-- Table: t_sys_role
CREATE TABLE IF NOT EXISTS "t_sys_role" (
  "id" integer NOT NULL,
  "name" text(16) NOT NULL,
  "code" text(64) NOT NULL,
  "status" integer(255),
  "remark" text(255),
  "type" integer(2) NOT NULL,
  "create_time"  datetime,
  "creator_id" integer(20),
  "creator" text(16),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(16),
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_sys_role (id, name, code, status, remark, type, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (7, '公共角色', 'COMMON', 1, '所有账号基础角色', 1, '2021-07-06 15:05:47', 1, 'admin', '2021-07-06 15:05:47', 1, 'admin', 0, NULL);
INSERT INTO t_sys_role (id, name, code, status, remark, type, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (8, '开发', 'DEV', 1, '研发人员', 0, '2021-07-09 10:46:10', 1, 'admin', '2021-07-09 10:46:10', 1, 'admin', 0, NULL);

-- Table: t_sys_role_resource
CREATE TABLE IF NOT EXISTS "t_sys_role_resource" (
  "id" integer NOT NULL,
  "role_id" integer(20) NOT NULL,
  "resource_id" integer(20) NOT NULL,
  "creator_id" integer(20),
  "creator" text(45),
  "create_time"  datetime,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (526, 7, 1, 1, 'admin', '2021-07-06 15:07:09', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (527, 8, 57, 1, 'admin', '2021-07-09 10:49:46', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (528, 8, 12, 1, 'admin', '2021-07-09 10:49:46', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (529, 8, 15, 1, 'admin', '2021-07-09 10:49:46', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (530, 8, 38, 1, 'admin', '2021-07-09 10:49:46', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (531, 8, 2, 1, 'admin', '2021-07-09 10:49:46', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (532, 8, 3, 1, 'admin', '2021-07-09 10:49:46', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (533, 8, 36, 1, 'admin', '2021-07-09 10:49:46', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (534, 8, 59, 1, 'admin', '2021-07-09 10:50:32', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (535, 7, 39, 1, 'admin', '2021-09-09 10:10:30', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (536, 8, 42, 1, 'admin', '2021-11-05 15:59:16', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (537, 8, 43, 1, 'admin', '2021-11-05 15:59:16', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (538, 8, 47, 1, 'admin', '2021-11-05 15:59:16', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (539, 8, 60, 1, 'admin', '2021-11-05 15:59:16', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (540, 8, 61, 1, 'admin', '2021-11-05 15:59:16', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (541, 8, 62, 1, 'admin', '2021-11-05 15:59:16', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (542, 8, 80, 1, 'admin', '2022-10-08 10:54:34', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (543, 8, 81, 1, 'admin', '2022-10-08 10:54:34', 0, NULL);
INSERT INTO t_sys_role_resource (id, role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (544, 8, 79, 1, 'admin', '2022-10-08 10:54:34', 0, NULL);

-- Table: t_tag_resource
CREATE TABLE IF NOT EXISTS "t_tag_resource" (
  "id" integer NOT NULL,
  "tag_id" integer(20) NOT NULL,
  "tag_path" text(255) NOT NULL,
  "resource_code" text(36),
  "resource_type" integer(4) NOT NULL,
  "create_time"  datetime NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(36) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "modifier" text(36) NOT NULL,
  "is_deleted" integer(4),
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);

-- Table: t_tag_tree
CREATE TABLE IF NOT EXISTS "t_tag_tree" (
  "id" integer NOT NULL,
  "pid" integer(20) NOT NULL,
  "code" text(36) NOT NULL,
  "code_path" text(255) NOT NULL,
  "name" text(36),
  "remark" text(255),
  "create_time"  datetime NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(36) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "modifier" text(36) NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_tag_tree (id, pid, code, code_path, name, remark, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (33, 0, 'default', 'default/', '默认', '默认标签', '2022-10-26 20:04:19', 1, 'admin', '2022-10-26 20:04:19', 1, 'admin', 0, NULL);

-- Table: t_tag_tree_team
CREATE TABLE IF NOT EXISTS "t_tag_tree_team" (
  "id" integer NOT NULL,
  "tag_id" integer(20) NOT NULL,
  "tag_path" text(255),
  "team_id" integer(20) NOT NULL,
  "create_time"  datetime NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(36) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "modifier" text(36) NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_tag_tree_team (id, tag_id, tag_path, team_id, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (31, 33, 'default/', 3, '2022-10-26 20:04:45', 1, 'admin', '2022-10-26 20:04:45', 1, 'admin', 0, NULL);

-- Table: t_team
CREATE TABLE IF NOT EXISTS "t_team" (
  "id" integer NOT NULL,
  "name" text(36) NOT NULL,
  "remark" text(255),
  "create_time"  datetime NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(36),
  "update_time"  datetime,
  "modifier_id" integer(20),
  "modifier" text(36),
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_team (id, name, remark, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (3, '默认团队', '默认团队', '2022-10-26 20:04:36', 1, 'admin', '2022-10-26 20:04:36', 1, 'admin', 0, NULL);

-- Table: t_team_member
CREATE TABLE IF NOT EXISTS "t_team_member" (
  "id" integer NOT NULL,
  "team_id" integer(20) NOT NULL,
  "account_id" integer(20) NOT NULL,
  "username" text(36) NOT NULL,
  "create_time"  datetime NOT NULL,
  "creator_id" integer(20) NOT NULL,
  "creator" text(36) NOT NULL,
  "update_time"  datetime NOT NULL,
  "modifier_id" integer(20) NOT NULL,
  "modifier" text(36) NOT NULL,
  "is_deleted" integer(8) NOT NULL,
  "delete_time"  datetime,
  PRIMARY KEY ("id")
);
INSERT INTO t_team_member (id, team_id, account_id, username, create_time, creator_id, creator, update_time, modifier_id, modifier, is_deleted, delete_time) VALUES (7, 3, 1, 'admin', '2022-10-26 20:04:36', 1, 'admin', '2022-10-26 20:04:36', 1, 'admin', 0, NULL);

-- Index: idx_db_backup_id
CREATE INDEX IF NOT EXISTS "idx_db_backup_id"
ON "t_db_backup_history" (
  "db_backup_id" ASC
);

-- Index: idx_db_instance_id
CREATE INDEX IF NOT EXISTS "idx_db_instance_id"
ON "t_db_backup" (
  "db_instance_id" ASC
);

-- Index: idx_db_name
CREATE INDEX IF NOT EXISTS "idx_db_name"
ON "t_db_backup" (
  "db_name" ASC
);

COMMIT TRANSACTION;
PRAGMA foreign_keys = on;
