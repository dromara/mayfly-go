cat /proc/uptime
echo '-----'
/bin/hostname -f
echo '-----'
cat /proc/loadavg
echo '-----'
cat /proc/meminfo
echo '-----'
df -B1
echo '-----'
/sbin/ip -o addr
echo '-----'
/bin/cat /proc/net/dev
echo '-----'
cat /proc/stat