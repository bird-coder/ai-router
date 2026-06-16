package middleware

import (
	"net/http"

	"github.com/bird-coder/manyo/pkg/logger"
	"github.com/gin-gonic/gin"
)

func RecoverHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("系统错误: %v", err)
				ctx.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		ctx.Next()
	}
}
