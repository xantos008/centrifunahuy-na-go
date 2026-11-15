package harness

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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

// ExpIn returns an exp claim value d from now (negative d gives an expired token).
// Expiry values are compared only through server replies, never as literals.
func ExpIn(d time.Duration) int64 { return time.Now().Add(d).Unix() }

var b64 = base64.RawURLEncoding

func unsigned(alg string, c Claims) string {
	payload, _ := json.Marshal(c)
	return b64.EncodeToString([]byte(`{"alg":"`+alg+`","typ":"JWT"}`)) + "." + b64.EncodeToString(payload)
}

func hs256(c Claims, secret []byte) string {
	u := unsigned("HS256", c)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(u))
	return u + "." + b64.EncodeToString(mac.Sum(nil))
}

// HS256 signs claims with the test HMAC secret.
func HS256(c Claims) string { return hs256(c, []byte(HMACSecret)) }

// HS256WithSecret signs with an arbitrary secret (invalid-signature cases).
func HS256WithSecret(c Claims, secret string) string { return hs256(c, []byte(secret)) }

// AlgConfusion is an HS256 token whose HMAC key is the RSA public key PEM, the
// classic attack on verifiers that pick the algorithm from the token header.
func AlgConfusion(c Claims) string { return hs256(c, []byte(RSAPublicKeyPEM)) }

// AlgNone is an unsigned token with "alg":"none".
func AlgNone(c Claims) string { return unsigned("none", c) + "." }

// RS256 signs with the test RSA key.
func RS256(c Claims) string {
	u := unsigned("RS256", c)
	h := sha256.Sum256([]byte(u))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey(rsaPrivateKeyPEM).(*rsa.PrivateKey), crypto.SHA256, h[:])
	if err != nil {
		panic(err)
	}
	return u + "." + b64.EncodeToString(sig)
}

// ES256 signs with the test P-256 key (JWS r||s signature encoding).
func ES256(c Claims) string {
	u := unsigned("ES256", c)
	h := sha256.Sum256([]byte(u))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey(ecdsaPrivateKeyPEM).(*ecdsa.PrivateKey), h[:])
	if err != nil {
		panic(err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return u + "." + b64.EncodeToString(sig)
}

// EdDSA signs with the test Ed25519 key.
func EdDSA(c Claims) string {
	u := unsigned("EdDSA", c)
	sig := ed25519.Sign(privateKey(ed25519PrivateKeyPEM).(ed25519.PrivateKey), []byte(u))
	return u + "." + b64.EncodeToString(sig)
}

func privateKey(p string) crypto.PrivateKey {
	block, _ := pem.Decode([]byte(p))
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic(err) // test-only constants; a parse error is a harness bug
	}
	return k
}
