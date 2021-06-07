

# 获取监控信息
function get_monitor_info() {
  cpu_rate=$(cat /proc/stat | awk '/cpu/{printf("%.2f%\n"), ($2+$4)*100/($2+$4+$5)}' | awk '{print $0}' | head -1)
  mem_rate=$(free -m | sed -n '2p' | awk '{print""($3/$2)*100"%"}')
  sys_load=$(uptime | cut -d: -f5)
  cat <<EOF | column -t
cpuRate:${cpu_rate},memRate:${mem_rate},sysLoad:${sys_load}
EOF
}

get_monitor_info

