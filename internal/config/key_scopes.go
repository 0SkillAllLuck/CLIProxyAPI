package config

import "strings"

// keyScopeProviderAliases maps the names people use for a provider to the
// provider key credentials are registered under.
var keyScopeProviderAliases = map[string]string{
	"anthropic": "claude",
	"openai":    "codex",
	"chatgpt":   "codex",
	"muse":      "meta",
	"grok":      "xai",
}

// keyScopeOpenAICompatiblePrefix starts the provider key an openai-compatibility
// entry registers its credentials under, as util.OpenAICompatibleProviderKey builds it.
const keyScopeOpenAICompatiblePrefix = "openai-compatible-"

// NormalizeKeyScopeProvider returns the provider key a provider name in
// client.key-scopes refers to: lower case, with aliases resolved.
func NormalizeKeyScopeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if canonical, ok := keyScopeProviderAliases[provider]; ok {
		return canonical
	}
	return provider
}

// KeyScopeProviderKeys returns the provider keys a provider name in
// client.key-scopes stands for: the provider it names, through an alias if it
// is one, and the key an openai-compatibility entry of that name registers its
// credentials under (openai-compatible-<name>).
func KeyScopeProviderKeys(provider string) []string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return nil
	}
	keys := []string{NormalizeKeyScopeProvider(provider)}
	if provider != "openai-compatibility" && !strings.HasPrefix(provider, keyScopeOpenAICompatiblePrefix) {
		keys = append(keys, keyScopeOpenAICompatiblePrefix+provider)
	}
	return keys
}

// KeyScope returns the client.key-scopes entry for a client API key: the one
// with the longest key-prefix the key starts with. Once any entry is configured,
// scoping fails closed: a key that matches no entry, or whose entry names no
// provider, gets an empty scope and may use no provider. ok is false only when no
// entry is configured or the request carries no key, and the key is unrestricted.
func (c ClientConfig) KeyScope(apiKey string) (scope ClientKeyScope, ok bool) {
	apiKey = strings.TrimSpace(apiKey)
	if c.KeyScopes == nil || len(*c.KeyScopes) == 0 || apiKey == "" {
		return ClientKeyScope{}, false
	}
	longest := -1
	for _, candidate := range *c.KeyScopes {
		prefix := strings.TrimSpace(candidate.KeyPrefix)
		if prefix == "" || len(prefix) <= longest || !strings.HasPrefix(apiKey, prefix) {
			continue
		}
		scope, longest = candidate, len(prefix)
	}
	if longest < 0 {
		return ClientKeyScope{}, true
	}
	return scope, true
}
