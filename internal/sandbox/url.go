package sandbox

import (
	"net/url"
	"strings"
)

// RewriteCallbackHost returns a copy of env with localhost references in URL
// values replaced by the given host. Non-URL values and values not containing
// localhost are left unchanged. The original map is not mutated.
func RewriteCallbackHost(env map[string]string, host string) map[string]string {
	if len(env) == 0 || host == "" {
		return env
	}

	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = rewriteURL(v, host)
	}
	return out
}

func rewriteURL(val, newHost string) string {
	if !isLocalhostURL(val) {
		return val
	}

	u, err := url.Parse(val)
	if err != nil {
		return val
	}

	port := u.Port()
	if port != "" {
		u.Host = newHost + ":" + port
	} else {
		u.Host = newHost
	}

	return u.String()
}

func isLocalhostURL(val string) bool {
	lower := strings.ToLower(val)
	return strings.HasPrefix(lower, "http://localhost") ||
		strings.HasPrefix(lower, "https://localhost") ||
		strings.HasPrefix(lower, "http://127.0.0.1") ||
		strings.HasPrefix(lower, "https://127.0.0.1")
}
