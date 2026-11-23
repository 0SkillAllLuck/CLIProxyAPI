package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// keyScopeMiddleware applies client.key-scopes to the routes that choose a
// credential without resolving a model. Codex web search and voice always use a
// Codex credential, so a key whose scope excludes codex is refused there before
// a credential is selected. Requests that name a model are scoped when the
// model is resolved, and every other route passes through unchanged.
func (s *Server) keyScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := singleProviderRoute(c.Request.Method, c.FullPath())
		if provider != "" && s.handlers != nil && !s.handlers.KeyAllowsProvider(c, provider) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{
				"message": provider + " is not available to this API key",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "provider_not_allowed",
			}})
			return
		}
		c.Next()
	}
}

// codexOnlyRoutes always use a Codex credential: Codex web search and voice.
// A Realtime client secret authenticates as the key that minted it, so the
// minting key's scope applies to it.
var codexOnlyRoutes = map[string]bool{
	"POST /v1/alpha/search":                         true,
	"POST /backend-api/codex/alpha/search":          true,
	"POST /v1/live":                                 true,
	"GET /v1/live/:call_id":                         true,
	"GET /v1/realtime":                              true,
	"POST /v1/realtime":                             true,
	"POST /v1/realtime/calls":                       true,
	"GET /v1/realtime/calls/:call_id":               true,
	"POST /v1/realtime/calls/:call_id/hangup":       true,
	"POST /v1/realtime/calls/:call_id/accept":       true,
	"POST /v1/realtime/calls/:call_id/reject":       true,
	"POST /v1/realtime/calls/:call_id/refer":        true,
	"POST /v1/realtime/client_secrets":              true,
	"POST /v1/realtime/sessions":                    true,
	"POST /v1/realtime/transcription_sessions":      true,
	"GET /v1/realtime/translations":                 true,
	"POST /v1/realtime/translations":                true,
	"POST /v1/realtime/translations/client_secrets": true,
}

func codexOnlyRoute(method, path string) bool { return codexOnlyRoutes[method+" "+path] }

// singleProviderRoute names the provider a route always uses without resolving
// a model: Codex search and voice, the Muse Code relay (a Meta credential) and
// the AI Studio websocket relay, where a client registers itself as an
// aistudio provider that other keys' requests can be routed to.
func singleProviderRoute(method, path string) string {
	switch {
	case codexOnlyRoute(method, path):
		return "codex"
	case strings.HasPrefix(path, "/muse-code/") && path != "/muse-code/models":
		return "meta"
	case path == "/v1/ws":
		return "aistudio"
	}
	return ""
}
