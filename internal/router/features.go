package router

import (
	"regexp"
	"strings"
)

type Request struct {
	Model string `json:"model"`

	// Codex / OpenAI responses API 输入
	Input any `json:"input"`

	// 工具（非常关键）
	Tools []Tool `json:"tools,omitempty"`

	// 可选参数
	MaxOutputTokens int     `json:"max_output_tokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`

	// 内部扩展（Router用）
	Metadata map[string]any `json:"metadata,omitempty"`
}

type Tool struct {
	Type     string `json:"type"` // function / file / etc
	Name     string `json:"name"`
	Function any    `json:"function,omitempty`
}

type chatMessage struct {
	Role    string           `json:"role"`
	Content []MessageContent `json:"content"`
}

type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Features struct {
	InputTokens     int
	HasTools        bool
	IsCodeTask      bool
	IsRefactor      bool
	IsSimpleQuery   bool
	RequiresLongCtx bool
	RequiresHighIQ  bool
}

func classify(req Request) Features {
	text := extractText(req.Input)
	return Features{
		InputTokens:     getInputTokens(text),
		HasTools:        checkHasTools(req.Tools),
		IsCodeTask:      checkIsCodeTask(text),
		IsRefactor:      checkIsRefactor(text),
		IsSimpleQuery:   checkIsSimpleQuery(text),
		RequiresLongCtx: checkRequiresLongCtx(text),
		RequiresHighIQ:  checkRequiresHighIQ(text),
	}
}

var (
	codePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*func\s+\w+`),    // Go
		regexp.MustCompile(`(?m)^\s*package\s+\w+`), // Go
		regexp.MustCompile(`(?m)^\s*import\s+`),     // 多语言
		regexp.MustCompile(`(?m)^\s*class\s+\w+`),   // Java/Python
		regexp.MustCompile(`(?m)^\s*def\s+\w+`),     // Python
		regexp.MustCompile(`(?m)^\s*#include\s+`),   // C/C++
		regexp.MustCompile(`(?m)^\s*<\w+>`),         // HTML/XML
		regexp.MustCompile("(?s)```.*?```"),         // Markdown code block
		regexp.MustCompile(`(?m)^\s*\w+\s*:=`),      // Go short assign
		regexp.MustCompile(`(?m)^\s*console\.log`),  // JS
	}
	keywords = []string{
		"func ", "package ", "import ", "class ",
		"def ", "return ", "if (", "for (",
		"```", "{", "};",
	}
)

func getInputTokens(text string) int {
	if len(text) == 0 {
		return 0
	}

	asciiCount := 0
	nonAsciiCount := 0

	for _, r := range text {
		if r < 128 {
			asciiCount++
		} else {
			nonAsciiCount++
		}
	}

	// 英文部分
	asciiTokens := asciiCount / 4
	// 中文部分
	nonAsciiTokens := nonAsciiCount

	// 乘一个安全系数
	return int(float64(asciiTokens+nonAsciiTokens) * 1.2)
}

func checkHasTools(tools []Tool) bool {
	return len(tools) > 0
}

func checkIsCodeTask(text string) bool {
	if len(text) == 0 {
		return false
	}

	// 关键字快速判断（性能更好）
	hits := 0
	for _, k := range keywords {
		if strings.Contains(text, k) {
			hits++
		}
	}

	if hits >= 2 {
		return true
	}

	// 正则兜底
	for _, p := range codePatterns {
		if p.MatchString(text) {
			return true
		}
	}

	return false
}

func checkIsRefactor(text string) bool {
	return strings.Contains(text, "refactor")
}

func checkIsSimpleQuery(text string) bool {
	return len(text) < 200
}

func checkRequiresLongCtx(text string) bool {
	tokens := getInputTokens(text)
	return tokens > 8000
}

func checkRequiresHighIQ(text string) bool {
	return strings.Contains(text, "design") || strings.Contains(text, "architecture")
}

func extractText(input any) string {
	switch v := input.(type) {
	case string:
		return v

	case []any:
		var sb strings.Builder

		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}

			content, ok := m["content"].([]any)
			if !ok {
				continue
			}

			for _, c := range content {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}

				if text, ok := cm["text"].(string); ok {
					sb.WriteString(text)
				}
			}
		}

		return sb.String()
	}

	return ""
}
