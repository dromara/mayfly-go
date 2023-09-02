package req

import (
	"errors"

	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 创建用户token
func CreateToken(userId uint64, username string) string {
	// 带权限创建令牌
	// 设置有效期，过期需要重新登录获取token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userId,
		"username": username,
		"exp":      time.Now().Add(time.Minute * time.Duration(config.Conf.Jwt.ExpireTime)).Unix(),
	})

	// 使用自定义字符串加密 and get the complete encoded token as a string
	tokenString, err := token.SignedString([]byte(config.Conf.Jwt.Key))
	biz.ErrIsNilAppendErr(err, "token创建失败: %s")
	return tokenString
}

// 解析token，并返回登录者账号信息
func ParseToken(tokenStr string) (*model.LoginAccount, error) {
	if tokenStr == "" {
		return nil, errors.New("token error")
	}

	// Parse token
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(config.Conf.Jwt.Key), nil
	})
	if err != nil || token == nil {
		return nil, err
	}
	i := token.Claims.(jwt.MapClaims)
	return &model.LoginAccount{Id: uint64(i["id"].(float64)), Username: i["username"].(string)}, nil
}
