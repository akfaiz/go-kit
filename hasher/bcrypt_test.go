package hasher_test

import (
	"strings"
	"testing"

	"github.com/akfaiz/go-kit/hasher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBcrypt_DefaultCost(t *testing.T) {
	h := hasher.NewBcrypt(hasher.BcryptConfig{})

	hash, err := h.Hash("password")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$2a$10$"))
}

func TestNewBcrypt_CustomCost(t *testing.T) {
	h := hasher.NewBcrypt(hasher.BcryptConfig{Cost: 4})

	hash, err := h.Hash("password")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$2a$04$"))
}

func TestBcryptHasher_Verify_InvalidHash(t *testing.T) {
	h := hasher.NewBcrypt(hasher.BcryptConfig{Cost: 4})

	ok, err := h.Verify("password", "not-a-bcrypt-hash")
	require.Error(t, err)
	assert.False(t, ok)
}
