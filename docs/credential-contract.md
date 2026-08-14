# Credential Contract & Privacy Posture

## Core Guarantees

The Meshery AI Adapter enforces these non-negotiable security guarantees:

### 1. Secrets Live Only in the Credential Store

Credentials (API keys, tokens) are stored exclusively in Meshery's Credential construct, which provides:
- **Encryption at rest** — secrets are encrypted before persistence
- **Access control** — credentials are scoped to the owning user
- **Audit trail** — CRUD operations are logged with operationId correlation

### 2. Secrets Are Never Returned to Clients

The `Credential` model uses `json:"-"` on the `Secret` field:

```go
type Credential struct {
    Secret map[string]string `json:"-"` // NEVER serialized
}
```

API responses return `CredentialResponse` with existence flags only:

```json
{
  "id": "abc-123",
  "name": "My OpenAI Key",
  "has_secret": { "api_key": true }
}
```

### 3. Secrets Are Never in Prompt Context

The `BuildPromptContext()` function constructs prompts from templates. It takes only the user's natural language intent and Meshery schema context. Credential values are never injected into prompts.

### 4. Secrets Are Never in Generated Designs

The `RedactSecrets()` function scans all LLM output for credential values before returning to clients. If an LLM accidentally echoes a key, it is replaced with `[REDACTED]`.

### 5. Secrets Are Never in Logs or Events

All log statements use safe projections. Health check messages are redacted before logging.

## What Data Is Sent to Providers

When using a hosted provider (OpenAI, Anthropic, Azure OpenAI):

| Data | Sent? | Notes |
|------|-------|-------|
| User's natural language prompt | ✅ | This is the user's intent |
| Meshery schema context | ✅ | Generic K8s component schemas |
| API key | ✅ | In HTTP auth header only, never in prompt body |
| Cluster secrets/data | ❌ | Never included |
| Other credentials | ❌ | Never included |
| User identity | ❌ | Never included |

## Local Inference (Ollama/LocalAI)

When using Ollama or LocalAI:
- **No data leaves the machine** — all processing is local
- **No API keys needed** — no credential storage required
- **Same generation quality** — depending on the model chosen
- **Ideal for air-gapped/compliance environments**

## Verification

Run the security test suite:

```bash
go test ./internal/ai/ -run "TestRedact|TestSecret" -v
```
