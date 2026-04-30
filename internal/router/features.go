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

func classify(text string) Features {
	return Features{
		IsRefactor: strings.Contains(text, "refactor"),
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

func getInputTokens() int {

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

func checkRequiresLongCtx() bool {

}

func checkRequiresHighIQ(text string) bool {
	return strings.Contains(text, "design") || strings.Contains(text, "architecture")
}
