#bin/bash

execfile=./mayfly-go

pid=`ps ax | grep -i 'mayfly-go' | grep -v grep | awk '{print $1}'`
if [ ! -z "${pid}" ] ; then
        echo "The mayfly-go already running, shutdown and restart..."
        kill ${pid}
fi

if [ ! -x "${execfile}" ]; then
  sudo chmod +x "${execfile}"
fi

nohup "${execfile}" &

echo "The mayfly-go running..."