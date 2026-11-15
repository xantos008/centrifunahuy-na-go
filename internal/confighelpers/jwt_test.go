// Modified by the Cedrazeka authors, 2026. See NOTICE.

package confighelpers

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/xantos008/centrifunahuy-na-go/internal/configtypes"

	"github.com/stretchr/testify/require"
)

func TestMakeVerifierConfigEd25519(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	cfg, err := MakeVerifierConfig(configtypes.Token{
		Ed25519PublicKey: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})),
	})
	require.NoError(t, err)
	require.True(t, pub.Equal(cfg.Ed25519PublicKey))

	_, err = MakeVerifierConfig(configtypes.Token{Ed25519PublicKey: "not a key"})
	require.ErrorContains(t, err, "Ed25519")
}
