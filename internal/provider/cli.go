package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ai-router/internal/config"
)

type CLI struct {
	cfg config.CLIProviderConfig
}

func NewCLI(cfg config.CLIProviderConfig) *CLI {
	return &CLI{cfg: cfg}
}

func (c *CLI) Run(ctx context.Context, req Request) (string, error) {
	tmpFile, err := os.CreateTemp("", "ai-router-cli-*.txt")
	if err != nil {
		return "", fmt.Errorf("create temp output: %w", err)
	}
	tmpName := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpName)

	command, args := c.renderCommand(req, tmpName)
	if command == "" {
		return "", fmt.Errorf("cli binary is required")
	}

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w: %s", command, err, strings.TrimSpace(string(output)))
	}

	if c.cfg.OutputMode == "stdout" {
		return strings.TrimSpace(string(output)), nil
	}

	data, err := os.ReadFile(tmpName)
	if err != nil {
		return "", fmt.Errorf("read cli output: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (c *CLI) renderCommand(req Request, outputFile string) (string, []string) {
	var args []string
	for _, arg := range c.cfg.Args {
		rendered := renderTemplate(arg, req, outputFile)
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		args = append(args, rendered)
	}
	return renderTemplate(c.cfg.Binary, req, outputFile), args
}

func renderTemplate(value string, req Request, outputFile string) string {
	replacer := strings.NewReplacer(
		"{{prompt}}", req.Prompt,
		"{{model}}", req.Model,
		"{{reasoning_effort}}", req.ReasoningEffort,
		"{{workdir}}", req.Workdir,
		"{{output_file}}", outputFile,
	)
	return replacer.Replace(value)
}
