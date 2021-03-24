# 获取系统cpu信息
function get_cpu_info() {
  Physical_CPUs=$(grep "physical id" /proc/cpuinfo | sort | uniq | wc -l)
  Virt_CPUs=$(grep "processor" /proc/cpuinfo | wc -l)
  CPU_Kernels=$(grep "cores" /proc/cpuinfo | uniq | awk -F ': ' '{print $2}')
  CPU_Type=$(grep "model name" /proc/cpuinfo | awk -F ': ' '{print $2}' | sort | uniq)
  CPU_Arch=$(uname -m)
  echo -e '\n-------------------------- CPU信息 --------------------------'
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
  sys_lastreboot=$(who -b | awk '{print $3,$4}')
  echo -e '-------------------------- 系统信息 --------------------------'
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
get_cpu_info