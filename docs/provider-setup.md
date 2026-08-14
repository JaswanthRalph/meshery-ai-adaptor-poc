# AI Provider Setup Guide

## OpenAI

### Prerequisites
- An OpenAI account with API access
- An API key from [platform.openai.com/api-keys](https://platform.openai.com/api-keys)

### Configuration

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `base_url` | No | `https://api.openai.com/v1` | API base URL |
| `model` | No | `gpt-4o` | Model to use for generation |
| `organization_id` | No | — | OpenAI organization ID |

### Credential

| Field | Required | Description |
|-------|----------|-------------|
| `api_key` | Yes | Your OpenAI API key (starts with `sk-`) |

### CLI Example
```bash
mesheryctl-ai connection create \
  --kind openai \
  --name "Production OpenAI" \
  --model gpt-4o \
  --api-key sk-proj-...
```

---

## Anthropic Claude

### Prerequisites
- An Anthropic account with API access
- An API key from [console.anthropic.com](https://console.anthropic.com)

### Configuration

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `base_url` | No | `https://api.anthropic.com` | API base URL |
| `model` | No | `claude-sonnet-4-20250514` | Model to use |
| `anthropic_version` | No | `2023-06-01` | API version header |

### Credential

| Field | Required | Description |
|-------|----------|-------------|
| `api_key` | Yes | Your Anthropic API key |

---

## Ollama (Local Inference)

### Prerequisites
- Ollama installed and running ([ollama.com](https://ollama.com))
- A model pulled: `ollama pull llama3.1`

### Configuration

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `base_url` | No | `http://localhost:11434` | Ollama server URL |
| `model` | No | `llama3.1` | Model name |

### Credential
**None required** — Ollama runs locally with no authentication.

### Privacy Advantage
With Ollama, your prompts and infrastructure data **never leave your machine**. This is ideal for:
- Air-gapped environments
- Sensitive infrastructure configurations
- Compliance-restricted organizations

### CLI Example
```bash
mesheryctl-ai connection create \
  --kind ollama \
  --name "Local Ollama" \
  --model llama3.1
```

---

## Azure OpenAI

### Prerequisites
- An Azure subscription with Azure OpenAI Service
- A deployed model in your Azure OpenAI resource

### Configuration

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `resource_name` | **Yes** | — | Azure resource name |
| `deployment_id` | **Yes** | — | Model deployment name |
| `api_version` | No | `2024-02-15-preview` | API version |

### Credential

| Field | Required | Description |
|-------|----------|-------------|
| `api_key` | Yes | Azure OpenAI API key |

### Enterprise Benefits
Azure OpenAI provides:
- Data residency guarantees
- Enterprise compliance (SOC2, HIPAA, etc.)
- Azure AD integration
- Private network deployment

---

## Google Cloud Vertex AI

### Prerequisites
- A Google Cloud Platform (GCP) project with the Vertex AI API enabled.
- A service account or an access token with permissions to invoke Vertex AI models.

### Configuration

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `project_id` | **Yes** | — | GCP Project ID |
| `location` | No | `us-central1` | GCP Region for the API |
| `model` | No | `gemini-1.5-pro` | Gemini model name |

### Credential

| Field | Required | Description |
|-------|----------|-------------|
| `access_token` | Yes | A valid GCP OAuth 2.0 access token |

### CLI Example
```bash
mesheryctl-ai connection create \
  --kind vertex-ai \
  --name "My Vertex AI" \
  --config project_id=my-gcp-project,location=us-east1,model=gemini-1.5-flash \
  --access-token ya29.c.c0...
```
