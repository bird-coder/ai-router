package httpapi

import (
	"ai-router/internal/service"
	"ai-router/internal/types"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	transferService *service.TransferService
}

func NewHandler(svc *service.TransferService) *Handler {
	return &Handler{
		transferService: svc,
	}
}

func (h Handler) HandleHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h Handler) HandleRoute(ctx *gin.Context) {
	var req types.GenerateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid json",
		})
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "prompt is required",
		})
		return
	}
	ctx.JSON(http.StatusBadRequest, gin.H{
		"error": "server stopped",
	})
	return
	resp, status := h.transferService.Transfer(ctx, req)
	ctx.JSON(status, resp)
}

func (h Handler) HandleOpenAIChatCompletions(ctx *gin.Context) {
	var req struct {
		Model    string              `json:"model"`
		Messages []types.ChatMessage `json:"messages"`
	}
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid json",
		})
		return
	}
	ctx.JSON(http.StatusBadRequest, gin.H{
		"error": "server stopped",
	})
	return

	prompt := flattenMessages(req.Messages)
	generateReq := types.GenerateRequest{
		Prompt:         prompt,
		PreferredModel: req.Model,
		Client:         "openai",
		TaskType:       inferTaskType(prompt),
	}
	resp, status := h.transferService.Transfer(ctx, generateReq)
	if resp.Error != "" {
		ctx.JSON(status, gin.H{
			"error": resp.Error,
		})
		return
	}

	jsonResp := types.OpenAIResponse{
		ID:      "chatcmpl-ai-router",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Route.Model,
		Choices: []types.Choice{
			{
				Index: 0,
				Message: types.ChatMessage{
					Role:    "assistant",
					Content: resp.Output,
				},
				FinishReason: "stop",
			},
		},
	}
	ctx.JSON(http.StatusOK, jsonResp)
}

func (h Handler) HandleAnthropicMessages(ctx *gin.Context) {
	var req struct {
		Model    string              `json:"model"`
		Messages []types.ChatMessage `json:"messages"`
	}
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid json",
		})
		return
	}
	ctx.JSON(http.StatusBadRequest, gin.H{
		"error": "server stopped",
	})
	return

	prompt := flattenMessages(req.Messages)
	generateReq := types.GenerateRequest{
		Prompt:         prompt,
		PreferredModel: req.Model,
		Client:         "anthropic",
		TaskType:       inferTaskType(prompt),
	}
	resp, status := h.transferService.Transfer(ctx, generateReq)
	if resp.Error != "" {
		ctx.JSON(status, gin.H{
			"error": resp.Error,
		})
		return
	}

	jsonResp := types.AnthropicResponse{
		ID:    "msg_ai_router",
		Type:  "message",
		Role:  "assistant",
		Model: resp.Route.Model,
		Content: []types.ChatContent{
			{
				Type: "text",
				Text: resp.Output,
			},
		},
		StopReason: "end_turn",
	}
	ctx.JSON(http.StatusOK, jsonResp)
}

func flattenMessages(messages []types.ChatMessage) string {
	var parts []string
	for _, message := range messages {
		text := flattenContent(message.Content)
		if strings.TrimSpace(text) == "" {
			continue
		}
		parts = append(parts, message.Role+": "+text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func flattenContent(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var parts []string
		for _, item := range typed {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			switch object["type"] {
			case "text":
				if text, ok := object["text"].(string); ok {
					parts = append(parts, text)
				}
			case "input_text":
				if text, ok := object["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func inferTaskType(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "review"):
		return "review"
	case strings.Contains(lower, "architecture"):
		return "architecture"
	case strings.Contains(lower, "fix"), strings.Contains(lower, "debug"):
		return "complex"
	default:
		return "fast"
	}
}
