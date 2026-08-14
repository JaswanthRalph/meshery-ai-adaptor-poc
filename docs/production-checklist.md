# AI Production Checklist

Use this checklist before enabling AI generation in a production Meshery deployment.

## ✅ Credential Security

- [ ] API keys are stored in Meshery Credentials (not environment variables)
- [ ] `MESHERY_AI_OPENAI_API_KEY` env var is removed from deployment configs
- [ ] Credential responses never contain secret values (verify with API test)
- [ ] Generated Designs are free of credential material (verify with redaction tests)
- [ ] Server logs contain no secret values (verify by grep)
- [ ] Event payloads contain no secret values

## ✅ Network & Egress

- [ ] Egress rules allow traffic to provider endpoints only:
  - OpenAI: `api.openai.com:443`
  - Anthropic: `api.anthropic.com:443`
  - Azure OpenAI: `*.openai.azure.com:443`
  - Ollama: `localhost:11434` (local only)
- [ ] No other outbound traffic is permitted from the AI subsystem
- [ ] TLS certificate validation is enabled (no `InsecureSkipVerify`)

## ✅ Provider Configuration

- [ ] At least one AI connection is configured and passing health checks
- [ ] Health check runs are scheduled or triggered on connection changes
- [ ] Provider timeout is set (default: 120s for hosted, 300s for local)
- [ ] Rate limiting is configured for the provider's tier

## ✅ Generation Safety

- [ ] Generation NEVER auto-deploys (verify: pipeline returns candidate only)
- [ ] Generated Designs pass schema validation before reaching the UI
- [ ] Dry-run validation against target cluster is available
- [ ] Users can review, edit, and explicitly approve before deployment
- [ ] Generation results include operationId for audit correlation

## ✅ Monitoring & Audit

- [ ] Generation requests are logged with: user, provider, model, latency, success
- [ ] Health check results are logged with operationId
- [ ] Connection CRUD operations are logged
- [ ] Credential creation/deletion is logged (without secret values)
- [ ] Alerts are configured for: auth failures, rate limiting, provider outages

## ✅ Credential Rotation

- [ ] Process exists for rotating API keys without downtime
- [ ] Old credentials can be deleted after rotation
- [ ] Rotation does not require server restart

## ✅ Privacy & Compliance

- [ ] Data handling documentation is reviewed and approved
- [ ] Local inference option (Ollama) is documented for air-gapped deployments
- [ ] Provider data processing agreements are in place (for hosted providers)
- [ ] Users understand what data is sent to providers (prompt + schema context only)

## Running the Checks

```bash
# Run all security and validation tests
go test ./internal/ai/... -v

# Verify no secrets in server logs
grep -ri "sk-\|api.key\|Bearer" /var/log/meshery/ || echo "✅ No secrets in logs"

# Verify API never returns secrets
curl -s http://localhost:9082/api/ai/credentials | jq '.[] | has("secret")'
# Should return: false (or no results)
```
