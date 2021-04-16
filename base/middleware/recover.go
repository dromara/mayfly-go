package middleware

import "github.com/gin-gonic/gin"

// RecoveryMiddleware 崩溃恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {

			}
		}()
		c.Next()
	}
}
