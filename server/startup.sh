#bin/bash

execfile=./mayfly-go

if [ ! -x "${execfile}" ]; then
  sudo chmod +x "${execfile}"
fi

nohup "${execfile}" &

echo "The mayfly-go running..."