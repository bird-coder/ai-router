package middleware

import (
	"net/http"

	"github.com/bird-coder/manyo/pkg/uniqid"
	"github.com/gin-gonic/gin"
)

func RequestId() gin.HandlerFunc {
	node, _ := uniqid.NewNode(1)
	return func(ctx *gin.Context) {
		if ctx.Request.Method == http.MethodOptions {
			ctx.Next()
			return
		}
		requestId := node.Generate()
		ctx.Set("RequestId", requestId.Int64())
		ctx.Next()
	}
}
