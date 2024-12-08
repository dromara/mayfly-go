Related configuration files: 
  backend:
    config.yml: Server port, mysql, aeskey(16, 24, 32 chars), jwt-key and other information can be configured here.
    It is recommended to replace aes.key(resource password encryption such as machine, database, redis password) and jwT.key (jwt secret key) with a random string.


server start & restart: ./startup.sh
server shutdown: ./shutdown.sh


The project can be accessed directly via host:port (port is server.port as configured in config.yml).
Initial account: admin/admin123.