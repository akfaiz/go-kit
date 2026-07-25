package hasher_test

import (
	"testing"

	"github.com/akfaiz/go-kit/hasher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_DefaultsToBcrypt(t *testing.T) {
	h, err := hasher.New()
	require.NoError(t, err)
	assert.IsType(t, &hasher.BcryptHasher{}, h)
}

func TestNew_Bcrypt(t *testing.T) {
	h, err := hasher.New(hasher.Config{Driver: hasher.DriverBcrypt})
	require.NoError(t, err)
	assert.IsType(t, &hasher.BcryptHasher{}, h)
}

func TestNew_Argon2id(t *testing.T) {
	h, err := hasher.New(hasher.Config{Driver: hasher.DriverArgon2id})
	require.NoError(t, err)
	assert.IsType(t, &hasher.Argon2idHasher{}, h)
}

func TestNew_UnsupportedDriver(t *testing.T) {
	h, err := hasher.New(hasher.Config{Driver: "md5"})
	require.Error(t, err)
	assert.Nil(t, h)
	assert.Contains(t, err.Error(), "md5")
}

func TestHasher_HashAndVerify(t *testing.T) {
	tests := []struct {
		name string
		cfg  hasher.Config
	}{
		{"bcrypt", hasher.Config{Driver: hasher.DriverBcrypt, Bcrypt: hasher.BcryptConfig{Cost: 4}}},
		{
			"argon2id",
			hasher.Config{
				Driver: hasher.DriverArgon2id,
				Argon2id: hasher.Argon2idConfig{
					Memory:      8 * 1024,
					Iterations:  1,
					Parallelism: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := hasher.New(tt.cfg)
			require.NoError(t, err)

			hash, err := h.Hash("s3cr3t-password")
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, "s3cr3t-password", hash)

			ok, err := h.Verify("s3cr3t-password", hash)
			require.NoError(t, err)
			assert.True(t, ok)

			ok, err = h.Verify("wrong-password", hash)
			require.NoError(t, err)
			assert.False(t, ok)
		})
	}
}

func TestHasher_HashProducesUniqueSalt(t *testing.T) {
	tests := []struct {
		name string
		cfg  hasher.Config
	}{
		{"bcrypt", hasher.Config{Driver: hasher.DriverBcrypt, Bcrypt: hasher.BcryptConfig{Cost: 4}}},
		{
			"argon2id",
			hasher.Config{
				Driver: hasher.DriverArgon2id,
				Argon2id: hasher.Argon2idConfig{
					Memory:      8 * 1024,
					Iterations:  1,
					Parallelism: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := hasher.New(tt.cfg)
			require.NoError(t, err)

			hash1, err := h.Hash("s3cr3t-password")
			require.NoError(t, err)
			hash2, err := h.Hash("s3cr3t-password")
			require.NoError(t, err)

			assert.NotEqual(t, hash1, hash2)
		})
	}
}
