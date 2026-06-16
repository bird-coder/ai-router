package server

import (
	"ai-router/internal/httpapi"
	"ai-router/internal/middleware"

	"github.com/bird-coder/manyo/config"
	"github.com/bird-coder/manyo/pkg/server/httpx"
	"github.com/gin-gonic/gin"
)

func NewHttp(cfg *config.HttpConfig, handler *httpapi.Handler) *httpx.HttpServer {
	httpServer := httpx.NewHttpServer(cfg)
	initRoutes(httpServer.Engine, handler)
	return httpServer
}

func initRoutes(g *gin.Engine, h *httpapi.Handler) {
	g.GET("health", h.HandleHealth)
	v1 := g.Group("v1")
	v1.Use(middleware.RecoverHandler(), middleware.LogHandler())
	{
		v1.POST("route", h.HandleRoute)
		v1.POST("responses", h.HandleOpenAIChatCompletions)
		v1.POST("messages", h.HandleAnthropicMessages)
	}
}
