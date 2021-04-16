module mayfly-go

go 1.16

require (
	// jwt
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-gonic/gin v1.6.2
	github.com/go-redis/redis v6.14.2+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gorilla/websocket v1.4.2
	github.com/onsi/ginkgo v1.16.1 // indirect
	github.com/onsi/gomega v1.11.0 // indirect
	github.com/pkg/sftp v1.12.0
	// 定时任务
	github.com/robfig/cron/v3 v3.0.1
	github.com/siddontang/go v0.0.0-20170517070808-cb568a3e5cc0
	github.com/sirupsen/logrus v1.6.0
	// ssh
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	gorm.io/driver/mysql v1.0.5
	gorm.io/gorm v1.21.6
)
