# 🌈mayfly-go

<p align="center">
  <a href="https://gitee.com/dromara/mayfly-go" target="_blank">
    <img src="https://gitee.com/dromara/mayfly-go/badge/star.svg?theme=white" alt="star"/>
    <img src="https://gitee.com/dromara/mayfly-go/badge/fork.svg" alt="fork"/>
  </a>
  <a href="https://github.com/dromara/mayfly-go" target="_blank">
    <img src="https://img.shields.io/github/stars/dromara/mayfly-go.svg?style=social" alt="github star"/>
    <img src="https://img.shields.io/github/forks/dromara/mayfly-go.svg?style=social" alt="github fork"/>
  </a>
  <a href="https://hub.docker.com/r/mayflygo/mayfly-go/tags" target="_blank">
    <img src="https://img.shields.io/docker/pulls/mayflygo/mayfly-go.svg?label=docker%20pulls&color=fac858" alt="docker pulls"/>
  </a>
  <a href="https://github.com/golang/go" target="_blank">
    <img src="https://img.shields.io/badge/Golang-1.22%2B-yellow.svg" alt="golang"/>
  </a>
  <a href="https://cn.vuejs.org" target="_blank">
    <img src="https://img.shields.io/badge/Vue-3.x-green.svg" alt="vue">
  </a>
</p>

### 介绍

web 版 **linux(终端[终端回放] 文件 脚本 进程 计划任务)、数据库（mysql postgres oracle sqlserver 达梦 高斯 sqlite）、redis(单机 哨兵 集群)、mongo 等集工单流程审批于一体的统一管理操作平台**

### 开发语言与主要框架

- 前端：typescript、vue3、element-plus
- 后端：golang、gin、gorm

### deploy with docker and sqlite
- 准备下载sqlite文件并初始化 [mayfly-go.sqlite](https://github.com/litongjava/mayfly-go/blob/master/server/resources/data/mayfly-go.sqlite)
- 准备下config.yml文件
```yml
server:
  # debug release test
  model: release
  port: 18888
  # 上下文路径, 若设置了该值, 则请求地址为ip:port/context-path
  # context-path: /mayfly
  cors: true
  tls:
    enable: false
    key-file: ./default.key
    cert-file: ./default.pem
jwt:
  # jwt key，不设置默认使用随机字符串
  key: 
  # 过期时间单位分钟
  expire-time: 1440
# 资源密码aes加密key
aes:
  key: 1111111111111111
# 若存在mysql配置，优先使用mysql
#mysql:
  # 自动升级数据库
#  auto-migration: true
#  host: 127.0.0.1:3306
#  username: root
#  password: 111049
#  db-name: mayfly-go
#  config: charset=utf8&loc=Local&parseTime=true
#  max-idle-conns: 5
sqlite:
  path: ./mayfly-go.sqlite
  max-idle-conns: 5
# 若同时部署多台机器，则需要配置redis信息用于缓存权限码、验证码、公私钥等
# redis:
#   host: localhost
#   port: 6379
#   password: 111049
#   db: 0
log:
  # 日志等级, debug, info, warn, error
  level: info
  # 日志格式类型, text/json
  type: text
  # 是否记录方法调用栈信息
  add-source: false
  # 日志文件配置
  # file:
  #   path: ./log
  #   name: mayfly-go.log
  #   # 日志文件的最大大小（以兆字节为单位）。当日志文件大小达到该值时，将触发切割操作
  #   max-size: 500
  #   # 根据文件名中的时间戳，设置保留旧日志文件的最大天数
  #   max-age: 60
  #   # 是否使用 gzip 压缩方式压缩轮转后的日志文件
  #   compress: true
```
执行部署命令
```shell
mkdir /data/docker/mayfly-go -p && cd /data/docker/mayfly-go
mkdir mayfly
cd mayfly
#add config.yml and 
cd ..
docker run -d --name mayfly-go -p 18888:18888 -e MAYFLY_JWT_KEY=53445c86e8189b6c646ed7d0d319015144423e72 -e MAYFLY_AES_KEY=7bc5418eefd50402ef39107274891fbe -v $(pwd)/mayfly:/mayfly litongjava/mayfly-go:v1.3.1
```
默认的用户和密码是admin/admin123.

### 交流及问题反馈加 QQ 群

<a target="_blank" href="https://qm.qq.com/cgi-bin/qm/qr?k=IdJSHW0jTMhmWFHBUS9a83wxtrxDDhFj&jump_from=webapi">119699946</a>

### 系统相关资料

- 项目文档: https://www.yuque.com/may-fly/mayfly-go
- 系统操作视频: https://space.bilibili.com/484091081/channel/collectiondetail?sid=392854

### 演示环境

http://go.mayfly.run
账号/密码：test/test123.

### 系统核心功能截图

##### 记录操作记录

![记录操作记录](https://objs.gitee.io/mayfly-go-docs/home/log.jpg "屏幕截图.png")

#### 机器操作

##### 状态查看

![状态查看](https://objs.gitee.io/mayfly-go-docs/home/machine-status.jpg "屏幕截图.png")

##### ssh 终端

![ssh终端](https://objs.gitee.io/mayfly-go-docs/home/machine-ssh.jpg "屏幕截图.png")

##### 文件操作

![文件操作](https://objs.gitee.io/mayfly-go-docs/home/file-dir.jpg "屏幕截图.png")
![文件操作](https://objs.gitee.io/mayfly-go-docs/home/file-content-update.jpg "屏幕截图.png")

#### 数据库操作

##### sql 编辑器

![sql编辑器](https://objs.gitee.io/mayfly-go-docs/home/dbms-sql-editor.jpg "屏幕截图.png")

##### 在线增删改查数据

![选表查数据](https://objs.gitee.io/mayfly-go-docs/home/dbms-show-table-data.jpg "屏幕截图.png")

#### Redis 操作

![数据](https://objs.gitee.io/mayfly-go-docs/home/redis-data-list.jpg "屏幕截图.png")

#### Mongo 操作

![数据](https://objs.gitee.io/mayfly-go-docs/home/mongo-op.jpg "屏幕截图.png")

##### 系统管理

##### 账号管理

![账号管理](https://images.gitee.com/uploads/images/2021/0607/173919_a8d7dc18_1240250.png "屏幕截图.png")

##### 角色管理

![角色管理](https://images.gitee.com/uploads/images/2021/0607/174028_3654fb28_1240250.png "屏幕截图.png")

##### 资源管理

![资源管理](https://images.gitee.com/uploads/images/2021/0607/174436_e9e1535c_1240250.png "屏幕截图.png")

**其他更多功能&操作指南可查看在线文档**: https://www.yuque.com/may-fly/mayfly-go

#### 💌 支持作者

如果觉得项目不错，或者已经在使用了，希望你可以去 <a target="_blank" href="https://github.com/dromara/mayfly-go">Github</a> 或者 <a target="_blank" href="https://gitee.com/dromara/mayfly-go">Gitee</a> 帮我点个 ⭐ Star，这将是对我极大的鼓励与支持。
