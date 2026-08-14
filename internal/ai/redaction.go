package ai

import (
	"strings"
)

// RedactSecrets scans input text and replaces any occurrence of
// secret values with "[REDACTED]". This is applied to:
//   - LLM responses before returning to clients
//   - Log output
//   - Event payloads
//   - Generated Design YAML
//
// This is a critical security measure: credentials must NEVER appear
// in any output that leaves the server.
func RedactSecrets(input string, secrets []string) string {
	if input == "" || len(secrets) == 0 {
		return input
	}
	result := input
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		// Replace all occurrences of the secret value
		result = strings.ReplaceAll(result, secret, "[REDACTED]")

		// Also redact partial matches for common patterns:
		// e.g., if key is "sk-abc123", also redact "sk-abc1..." truncations
		if len(secret) > 8 {
			prefix := secret[:8]
			// Only redact if the prefix appears with trailing chars
			// This catches cases where the LLM might have memorized partial keys
			if strings.Contains(result, prefix) {
				// Find and redact any string that starts with the prefix
				// and looks like a key (alphanumeric + dashes)
				result = redactPrefixMatches(result, prefix)
			}
		}
	}
	return result
}

// redactPrefixMatches finds strings starting with prefix that look like API keys
// and replaces them with [REDACTED].
func redactPrefixMatches(input, prefix string) string {
	result := []byte(input)
	prefixBytes := []byte(prefix)

	for i := 0; i < len(result); i++ {
		if i+len(prefixBytes) > len(result) {
			break
		}
		// Check if this position matches the prefix
		match := true
		for j := range prefixBytes {
			if result[i+j] != prefixBytes[j] {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		// Find the end of the key-like string
		end := i + len(prefixBytes)
		for end < len(result) && isKeyChar(result[end]) {
			end++
		}

		// Only redact if the match is substantially longer than the prefix
		// (to avoid false positives on short common strings)
		if end-i > len(prefixBytes)+4 {
			replacement := []byte("[REDACTED]")
			newResult := make([]byte, 0, len(result)-end+i+len(replacement))
			newResult = append(newResult, result[:i]...)
			newResult = append(newResult, replacement...)
			newResult = append(newResult, result[end:]...)
			result = newResult
		}
	}
	return string(result)
}

// isKeyChar returns true if a byte is a valid character in an API key.
func isKeyChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '-' || b == '_' || b == '.'
}

// CollectSecrets extracts all secret values from a credential
// for use with RedactSecrets.
func CollectSecrets(secrets map[string]string) []string {
	if secrets == nil {
		return nil
	}
	values := make([]string, 0, len(secrets))
	for _, v := range secrets {
		if v != "" {
			values = append(values, v)
		}
	}
	return values
}
