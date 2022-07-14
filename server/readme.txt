相关配置文件: 
  后端：
    config.yml: 服务端口，mysql等信息在此配置即可。
  前端：
    static/config.js: 若前后端分开部署则将该文件中的api地址配成后端服务的真实地址即可，否则无需修改。

服务启动：./startup.sh
服务关闭：./shutdown.sh

直接通过 host:ip即可访问项目
初始账号 admin/123456