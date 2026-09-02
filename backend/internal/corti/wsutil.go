package corti

import (
	"net/url"
	"time"

	gorillaws "github.com/gorilla/websocket"
)

// newCortiDialer returns the WebSocket dialer both proxies (ambient,
// dictation) connect to Corti with — same handshake timeout, no
// compression, in both cases.
func newCortiDialer() gorillaws.Dialer {
	return gorillaws.Dialer{
		HandshakeTimeout:  30 * time.Second,
		EnableCompression: false,
	}
}

// encodeBearerToken URL-encodes an access token as the "Bearer <token>"
// value Corti's WebSocket endpoints expect in the query string.
func encodeBearerToken(token string) string {
	return url.QueryEscape("Bearer " + token)
}

// primaryLanguageCode reduces a language tag like "en-US" to its 2-letter
// primary subtag ("en"), which is what Corti's stream config expects.
func primaryLanguageCode(lang string) string {
	if len(lang) > 2 && lang[2] == '-' {
		return lang[:2]
	}
	return lang
}
