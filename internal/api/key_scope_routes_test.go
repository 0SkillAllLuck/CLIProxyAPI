package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
)

// TestUntrustedKeyScopeOnServer runs the real route table with a key limited to
// Kimi and an openai-compatibility GLM entry, next to a key for the subscription
// providers, and checks every request shape refuses the subscription models.
func TestUntrustedKeyScopeOnServer(t *testing.T) {
	const claudeModel, kimiModel = "scope-e2e-claude", "scope-e2e-kimi"
	modelRegistry := registry.GetGlobalRegistry()
	modelRegistry.RegisterClient("scope-e2e-claude", "claude", []*registry.ModelInfo{{ID: claudeModel}})
	modelRegistry.RegisterClient("scope-e2e-kimi", "kimi", []*registry.ModelInfo{{ID: kimiModel}})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient("scope-e2e-claude")
		modelRegistry.UnregisterClient("scope-e2e-kimi")
	})

	const untrusted, personal = "sk-untrusted-1", "sk-personal-1"
	cfg := &config.Config{SDKConfig: config.SDKConfig{APIKeys: []string{untrusted, personal}}}
	cfg.WebsocketAuth = true
	cfg.Client.KeyScopes = &[]config.ClientKeyScope{
		{KeyPrefix: "sk-untrusted-", Providers: []string{"kimi", "glm"}},
		{KeyPrefix: "sk-personal-", Providers: []string{"claude", "codex", "antigravity", "muse", "grok"}},
	}
	server := newTestServerWithConfig(t, cfg)
	server.AttachWebsocketRoute("/v1/ws", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	serve := func(method, path, key, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+key)
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		server.engine.ServeHTTP(recorder, request)
		return recorder
	}
	refused := func(r *httptest.ResponseRecorder) bool {
		return r.Code == http.StatusBadRequest && strings.Contains(r.Body.String(), "not available to this API key")
	}

	modelRoutes := []struct{ path, body string }{
		{"/v1/messages", `{"model":"%s","max_tokens":8,"messages":[{"role":"user","content":"hi"}]}`},
		{"/v1/messages/count_tokens", `{"model":"%s","messages":[{"role":"user","content":"hi"}]}`},
		{"/v1/chat/completions", `{"model":"%s","messages":[{"role":"user","content":"hi"}]}`},
		{"/v1/responses", `{"model":"%s","input":"hi"}`},
	}
	for _, route := range modelRoutes {
		for _, tc := range []struct {
			key, model string
			want       bool
		}{
			{untrusted, claudeModel, true},
			{untrusted, kimiModel, false},
			{personal, claudeModel, false},
			{personal, kimiModel, true},
		} {
			// Allowed requests fail later for want of a credential; only the scope check is under test.
			if r := serve(http.MethodPost, route.path, tc.key, strings.ReplaceAll(route.body, "%s", tc.model)); refused(r) != tc.want {
				t.Errorf("%s %s with %s: status=%d body=%s, want refused=%t", route.path, tc.model, tc.key, r.Code, r.Body.String(), tc.want)
			}
		}
	}

	for _, ua := range []string{"", "claude-cli/2.0"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		request.Header.Set("Authorization", "Bearer "+untrusted)
		request.Header.Set("User-Agent", ua)
		recorder := httptest.NewRecorder()
		server.engine.ServeHTTP(recorder, request)
		if body := recorder.Body.String(); recorder.Code != http.StatusOK || strings.Contains(body, claudeModel) || !strings.Contains(body, kimiModel) {
			t.Errorf("/v1/models (UA %q) for untrusted key: status=%d body=%s", ua, recorder.Code, body)
		}
	}
	if r := serve(http.MethodGet, "/v1/models/"+claudeModel, untrusted, ""); r.Code != http.StatusNotFound {
		t.Errorf("/v1/models/{claude} for untrusted key: status=%d", r.Code)
	}

	// Routes that always use one provider's credential without naming a model.
	for _, route := range []struct{ method, path string }{
		{http.MethodPost, "/v1/alpha/search"},
		{http.MethodPost, "/v1/realtime/client_secrets"},
		{http.MethodPost, "/muse-code/search"},
		{http.MethodGet, "/muse-code/config"},
		{http.MethodGet, "/v1/ws"},
	} {
		if r := serve(route.method, route.path, untrusted, ""); r.Code != http.StatusForbidden || keyScopeErrorCode(r.Body.Bytes()) != "provider_not_allowed" {
			t.Errorf("%s %s with untrusted key: status=%d body=%s", route.method, route.path, r.Code, r.Body.String())
		}
	}
	if r := serve(http.MethodGet, "/muse-code/config", personal, ""); r.Code == http.StatusForbidden {
		t.Errorf("/muse-code/config with personal key: status=%d body=%s", r.Code, r.Body.String())
	}
}
