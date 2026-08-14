// Copyright 2026 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ai

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ContextBuilder constructs the system prompt and schema context
// that ground the LLM's output in valid Meshery Design format.
//
// The context is kept compact to fit within model token limits
// while providing enough schema information for valid output.

// SystemPrompt is the base system prompt for all AI providers.
const SystemPrompt = `You are Meshery AI, an expert infrastructure architect.
Your task is to generate valid Meshery Design documents from natural language descriptions.

## Output Format
You MUST respond with a valid JSON object matching the Meshery Design schema.
Do NOT include any explanation, markdown, or code fences. Output ONLY the JSON.

## Design Schema
A Meshery Design has this structure:
{
  "name": "design-name",
  "schema_version": "designs.meshery.io/v1beta1",
  "version": "1.0.0",
  "components": [
    {
      "name": "component-name",
      "kind": "Deployment|Service|ConfigMap|Ingress|...",
      "apiVersion": "apps/v1|v1|networking.k8s.io/v1|...",
      "model": "kubernetes",
      "namespace": "default",
      "labels": {"app": "name"},
      "config": {
        // Kubernetes resource spec fields
      }
    }
  ],
  "relationships": [
    {
      "kind": "edge",
      "type": "network|mount|binding",
      "source": "source-component-name",
      "target": "target-component-name"
    }
  ]
}

## Rules
1. Every component MUST have a valid Kubernetes kind and apiVersion
2. Deployments must specify replicas, container image, and ports in config
3. Services must specify type, port, and targetPort in config
4. Use sensible defaults when the user doesn't specify details
5. Always include labels for component linking
6. Create appropriate relationships between components
7. NEVER include any secrets, API keys, passwords, or credentials in the config
8. Namespace defaults to "default" unless specified`

// SchemaContext provides compact Kubernetes component schemas
// for the LLM to reference when generating Designs.
const SchemaContext = `## Available Kubernetes Components (Compact Schema)

### Deployment (apps/v1)
config: {replicas: int, containers: [{name, image, ports: [{containerPort}], env: [{name, value}], resources: {limits: {cpu, memory}, requests: {cpu, memory}}}]}

### Service (v1)
config: {type: "ClusterIP|NodePort|LoadBalancer", ports: [{port, targetPort, protocol}], selector: {app: "name"}}

### ConfigMap (v1)
config: {data: {"key": "value"}}

### Ingress (networking.k8s.io/v1)
config: {rules: [{host, http: {paths: [{path, pathType, backend: {service: {name, port: {number}}}}]}}]}

### PersistentVolumeClaim (v1)
config: {accessModes: ["ReadWriteOnce"], resources: {requests: {storage: "10Gi"}}, storageClassName: "standard"}

### HorizontalPodAutoscaler (autoscaling/v2)
config: {scaleTargetRef: {apiVersion, kind, name}, minReplicas, maxReplicas, metrics: [{type, resource: {name, target: {type, averageUtilization}}}]}

### Namespace (v1)
config: {}`

// FetchDynamicSchemas attempts to pull real schemas from a local Meshery server.
// If it fails (e.g. server not running), it falls back to the hardcoded SchemaContext.
func FetchDynamicSchemas() string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:9081/api/meshmodels/models?pagesize=1")
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		// We've successfully connected to Meshery. In a full integration, we would parse
		// the JSON and extract the actual CRD schemas. For this PoC, we append a notice.
		return SchemaContext + "\n\n(Dynamic Schema Note: Connection to Meshery established. Real schemas would be injected here.)"
	}
	return SchemaContext
}

// BuildPromptContext creates the full context for a generation request.
func BuildPromptContext(userPrompt string) *GenerateInput {
	return &GenerateInput{
		UserPrompt:    userPrompt,
		SystemPrompt:  SystemPrompt,
		SchemaContext: FetchDynamicSchemas(),
		JSONMode:      true, // Enable JSON mode by default
	}
}

// BuildPromptContextWithModel creates context with a model override.
func BuildPromptContextWithModel(userPrompt, model string) *GenerateInput {
	input := BuildPromptContext(userPrompt)
	input.Model = model
	return input
}

// FewShotExamples provides example prompt→Design pairs for better grounding.
var FewShotExamples = []struct {
	Prompt string
	Design string
}{
	{
		Prompt: "Deploy a simple nginx web server",
		Design: fmt.Sprintf(`{
  "name": "nginx-simple",
  "schema_version": "designs.meshery.io/v1beta1",
  "version": "1.0.0",
  "components": [
    {
      "name": "nginx-deployment",
      "kind": "Deployment",
      "apiVersion": "apps/v1",
      "model": "kubernetes",
      "namespace": "default",
      "labels": {"app": "nginx"},
      "config": {
        "replicas": 1,
        "containers": [{"name": "nginx", "image": "nginx:latest", "ports": [{"containerPort": 80}]}]
      }
    },
    {
      "name": "nginx-service",
      "kind": "Service",
      "apiVersion": "v1",
      "model": "kubernetes",
      "namespace": "default",
      "labels": {"app": "nginx"},
      "config": {
        "type": "ClusterIP",
        "ports": [{"port": 80, "targetPort": 80, "protocol": "TCP"}],
        "selector": {"app": "nginx"}
      }
    }
  ],
  "relationships": [
    {"kind": "edge", "type": "network", "source": "nginx-service", "target": "nginx-deployment"}
  ]
}`),
	},
}

// EnhanceSystemPromptWithExamples adds few-shot examples to the system prompt.
func EnhanceSystemPromptWithExamples(base string) string {
	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n## Examples\n")
	for _, ex := range FewShotExamples {
		sb.WriteString(fmt.Sprintf("\nPrompt: %s\nResponse:\n%s\n", ex.Prompt, ex.Design))
	}
	return sb.String()
}
