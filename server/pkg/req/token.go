package req

import (
	"errors"
	"mayfly-go/pkg/utils/stringx"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JwtConf jwt配置
type JwtConf struct {
	Key                    string
	ExpireTime             uint64 // 过期时间，单位分钟
	RefreshTokenExpireTime uint64 // 刷新token的过期时间，单位分钟
}

// 默认jwt配置
var jwtConf = JwtConf{
	Key:                    stringx.RandUUID(),
	ExpireTime:             60,
	RefreshTokenExpireTime: 360,
}

// SetJwtConf 设置jwt配置
func SetJwtConf(conf JwtConf) {
	jwtConf = conf
}

// 创建用户token
func CreateToken(userId uint64, username string) (accessToken string, refreshToken string, err error) {
	now := time.Now()

	// 带权限创建令牌
	// 设置有效期，过期需要重新登录获取token
	accessJwt := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userId,
		"username": username,
		"exp":      now.Add(time.Minute * time.Duration(jwtConf.ExpireTime)).Unix(),
	})

	// refresh token
	refreshJwt := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userId,
		"username": username,
		"exp":      now.Add(time.Minute * time.Duration(jwtConf.RefreshTokenExpireTime)).Unix(),
	})

	// 使用自定义字符串加密 and get the complete encoded token as a string
	accessToken, err = accessJwt.SignedString([]byte(jwtConf.Key))
	if err != nil {
		return
	}

	refreshToken, err = refreshJwt.SignedString([]byte(jwtConf.Key))
	return
}

// 解析token，并返回登录者账号信息
func ParseToken(tokenStr string) (uint64, string, error) {
	if tokenStr == "" {
		return 0, "", errors.New("token error")
	}

	// Parse token
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(jwtConf.Key), nil
	})
	if err != nil || token == nil {
		return 0, "", err
	}
	if !token.Valid {
		return 0, "", errors.New("token invalid")
	}
	i := token.Claims.(jwt.MapClaims)
	return uint64(i["id"].(float64)), i["username"].(string), nil
}
