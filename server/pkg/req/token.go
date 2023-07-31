package req

import (
	"errors"

	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 初始化jwt key与expire time等
func InitTokenConfig() {
	if ExpTime == 0 {
		JwtKey = config.Conf.Jwt.Key
		ExpTime = config.Conf.Jwt.ExpireTime

		// 如果配置文件中的jwt key为空，则随机生成字符串
		if JwtKey == "" {
			JwtKey = stringx.Rand(32)
			global.Log.Infof("config.yml未配置jwt.key, 随机生成key为: %s", JwtKey)
		}
	}
}

var (
	JwtKey  string
	ExpTime uint64
)

// 创建用户token
func CreateToken(userId uint64, username string) string {
	InitTokenConfig()

	// 带权限创建令牌
	// 设置有效期，过期需要重新登录获取token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userId,
		"username": username,
		"exp":      time.Now().Add(time.Minute * time.Duration(ExpTime)).Unix(),
	})

	// 使用自定义字符串加密 and get the complete encoded token as a string
	tokenString, err := token.SignedString([]byte(JwtKey))
	biz.ErrIsNilAppendErr(err, "token创建失败: %s")
	return tokenString
}

// 解析token，并返回登录者账号信息
func ParseToken(tokenStr string) (*model.LoginAccount, error) {
	if tokenStr == "" {
		return nil, errors.New("token error")
	}
	InitTokenConfig()

	// Parse token
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(JwtKey), nil
	})
	if err != nil || token == nil {
		return nil, err
	}
	i := token.Claims.(jwt.MapClaims)
	return &model.LoginAccount{Id: uint64(i["id"].(float64)), Username: i["username"].(string)}, nil
}
