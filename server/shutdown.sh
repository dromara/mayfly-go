#bin/bash

pid=`ps ax | grep -i 'mayfly-go' | grep -v grep | awk '{print $1}'`
if [ -z "${pid}" ] ; then
        echo "No mayfly-go running."
        exit -1;
fi

echo "The mayfly-go(${pid}) is running..."

kill ${pid}

echo "Send shutdown request to mayfly-go(${pid}) OK"