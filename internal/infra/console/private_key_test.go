package console

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuggestedPrivateKey(t *testing.T) {
	t.Parallel()

	t.Run(`Given a keys folder with a single private key
	When the private key is suggested
	Then it is that key`, func(t *testing.T) {
		t.Parallel()
		folder := t.TempDir()
		keys := writeKeyPair(t, folder, "1234")

		require.Equal(t, keys.privateKey, suggestedPrivateKey(folder))
	})

	t.Run(`Given a keys folder with several private keys
	When the private key is suggested
	Then it is the folder, to complete with Tab`, func(t *testing.T) {
		t.Parallel()
		folder := t.TempDir()
		writeKeyPair(t, folder, "1234")
		writeKeyPair(t, folder, "5678")

		require.Equal(t, folder+string(filepath.Separator), suggestedPrivateKey(folder))
	})
}
