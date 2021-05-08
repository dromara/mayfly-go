package ctx

import (
	"errors"

	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	JwtKey  = "mykey"
	ExpTime = time.Hour * 24 * 7
)

// 创建用户token
func CreateToken(userId uint64, username string) string {
	// 带权限创建令牌
	// 设置有效期，过期需要重新登录获取token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userId,
		"username": username,
		"exp":      time.Now().Add(ExpTime).Unix(),
	})

	// 使用自定义字符串加密 and get the complete encoded token as a string
	tokenString, err := token.SignedString([]byte(JwtKey))
	biz.ErrIsNil(err, "token创建失败")
	return tokenString
}

// 解析token，并返回登录者账号信息
func ParseToken(tokenStr string) (*model.LoginAccount, error) {
	if tokenStr == "" {
		return nil, errors.New("token error")
	}
	// Parse token
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(JwtKey), nil
	})
	if err != nil || token == nil {
		return nil, err
	}
	i := token.Claims.(jwt.MapClaims)
	return &model.LoginAccount{Id: uint64(i["id"].(float64)), Username: i["username"].(string)}, nil
}
