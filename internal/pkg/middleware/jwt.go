package middleware

import (
	"strings"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const userIDKey = "userID"

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer"))
		claims := jwt.MapClaims{}
		t, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !t.Valid {
			response.Fail(c, errs.ErrUnauthorized)
			c.Abort()
			return
		}
		uid, ok := claims["uid"].(float64)
		if !ok {
			response.Fail(c, errs.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set(userIDKey, int64(uid))
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) int64 {
	v, _ := c.Get(userIDKey)
	id, _ := v.(int64)
	return id
}
