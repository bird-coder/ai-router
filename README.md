# ai-router

`ai-router` is a small HTTP gateway that routes incoming AI tasks to different providers and models based on simple rules.

## Current scope

- Exposes `POST /v1/route`
- Exposes OpenAI-style `POST /v1/chat/completions`
- Exposes Anthropic-style `POST /v1/messages`
- Resolves a target provider and model from request metadata and local route rules
- Supports local CLI backends such as Codex and Claude Code
- Supports OpenAI-compatible HTTP backends such as Qwen, MiniMax, and ChatGPT

## Project layout

```text
cmd/server/main.go          entrypoint
internal/config             config loading
internal/httpapi            HTTP handlers
internal/router             rule-based model selection
internal/provider           CLI/HTTP adapters and provider registry
internal/types              request and response payloads
config.example.json         sample routing config
```

## Run

```bash
cd /Users/jiajie.yu/go/src/ai-router
AI_ROUTER_CONFIG=config.example.json go run ./cmd/server
```

## Generic route request

```bash
curl -s http://localhost:8080/v1/route \
  -H 'Content-Type: application/json' \
  -d '{
    "prompt": "Reply with exactly: pong",
    "task_type": "fast",
    "timeout_seconds": 60
  }'
```

## Example response

```json
{
  "route": {
    "rule_name": "fast-default",
    "provider": "qwen",
    "model": "qwen-plus",
    "reasoning_effort": "low",
    "resolved_workdir": "/Users/jiajie.yu/go/src"
  },
  "output": "pong"
}
```

## Compatibility endpoints

OpenAI-style:

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-5.4",
    "messages": [{"role":"user","content":"review this code"}]
  }'
```

Anthropic-style:

```bash
curl -s http://localhost:8080/v1/messages \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "claude-sonnet-4",
    "messages": [{"role":"user","content":"summarize this diff"}]
  }'
```

## Notes

- `claude` / `claude-code` is not bundled here; configure the CLI provider after installing it locally.
- The example MiniMax configuration is only a placeholder and may need vendor-specific request shaping before production use.

## Next steps

- Add per-provider request shapers for vendors that are not fully OpenAI-compatible
- Add auth, rate limiting, and request logging
- Add rule sources from database or admin API
- Add async job mode for long-running requests
