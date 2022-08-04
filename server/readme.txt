相关配置文件: 
  后端：
    config.yml: 服务端口，mysql，aeskey(16 24 32位)，jwtkey等信息在此配置即可。
    建议务必将aes.key(资源密码加密如机器、数据库、redis等密码)与jwt.key(jwt秘钥)两信息使用随机字符串替换。

  前端：
    static/config.js: 若前后端分开部署则将该文件中的api地址配成后端服务的真实地址即可，否则无需修改。

服务启动&重启：./startup.sh
服务关闭：./shutdown.sh

直接通过 host:ip即可访问项目
初始账号 admin/admin123.