module mayfly-go

go 1.16

require (
	// jwt
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-gonic/gin v1.7.2
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/gorilla/websocket v1.4.2
	// 验证码
	github.com/mojocn/base64Captcha v1.3.4
	github.com/onsi/ginkgo v1.16.4 // indirect
	github.com/onsi/gomega v1.13.0 // indirect
	github.com/pkg/sftp v1.13.1
	// 定时任务
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.8.1
    // jsonschemal校验
	github.com/xeipuuv/gojsonschema v1.2.0
	// ssh
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	// gorm
	gorm.io/driver/mysql v1.0.5
	gorm.io/gorm v1.21.11
)
