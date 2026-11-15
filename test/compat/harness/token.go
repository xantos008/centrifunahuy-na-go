package harness

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

// Claims is the subset of connection/subscription JWT claims the scenarios use.
type Claims struct {
	Sub     string `json:"sub"`
	Exp     int64  `json:"exp,omitempty"`
	Channel string `json:"channel,omitempty"`
	Info    *Info  `json:"info,omitempty"`
}

// Info is connection info attached to presence and publications.
type Info struct {
	Name string `json:"name"`
}

// HS256 signs claims with the test HMAC secret.
func HS256(c Claims) string {
	return signHS256(c, HMACSecret)
}

// HS256WithSecret signs with an arbitrary secret (for invalid-signature cases).
func HS256WithSecret(c Claims, secret string) string { return signHS256(c, secret) }

// ExpIn returns an exp claim value d from now. Expiry values are compared
// only through server error codes, never as literal values.
func ExpIn(d time.Duration) int64 { return time.Now().Add(d).Unix() }

func signHS256(c Claims, secret string) string {
	enc := base64.RawURLEncoding
	header := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(c)
	unsigned := header + "." + enc.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + enc.EncodeToString(mac.Sum(nil))
}
