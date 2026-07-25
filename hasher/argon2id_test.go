package hasher_test

import (
	"strings"
	"testing"

	"github.com/akfaiz/go-kit/hasher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testArgon2idConfig() hasher.Argon2idConfig {
	return hasher.Argon2idConfig{
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
	}
}

func TestNewArgon2id_EncodesPHCFormat(t *testing.T) {
	h := hasher.NewArgon2id(testArgon2idConfig())

	hash, err := h.Hash("password")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$m=8192,t=1,p=1$"))

	parts := strings.Split(hash, "$")
	require.Len(t, parts, 6)
}

func TestArgon2idHasher_Verify_InvalidHashFormat(t *testing.T) {
	h := hasher.NewArgon2id(testArgon2idConfig())

	tests := []struct {
		name string
		hash string
	}{
		{"not enough parts", "$argon2id$v=19$m=8192,t=1,p=1$salt"},
		{"wrong algorithm", "$bcrypt$v=19$m=8192,t=1,p=1$salt$hash"},
		{"bad version", "$argon2id$v=abc$m=8192,t=1,p=1$c2FsdA$aGFzaA"},
		{"unsupported version", "$argon2id$v=1$m=8192,t=1,p=1$c2FsdA$aGFzaA"},
		{"bad params", "$argon2id$v=19$m=x,t=1,p=1$c2FsdA$aGFzaA"},
		{"bad salt", "$argon2id$v=19$m=8192,t=1,p=1$not base64!$aGFzaA"},
		{"bad hash", "$argon2id$v=19$m=8192,t=1,p=1$c2FsdA$not base64!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := h.Verify("password", tt.hash)
			require.Error(t, err)
			assert.False(t, ok)
		})
	}
}
