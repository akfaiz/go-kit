package hasher

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idConfig configures an Argon2idHasher. Zero-value fields default to
// OWASP-recommended parameters.
type Argon2idConfig struct {
	// Memory is the memory cost in KiB. Defaults to 64*1024 (64 MiB) when zero.
	Memory uint32
	// Iterations is the number of passes over memory. Defaults to 3 when zero.
	Iterations uint32
	// Parallelism is the number of parallel threads. Defaults to 2 when zero.
	Parallelism uint8
	// SaltLength is the random salt size in bytes. Defaults to 16 when zero.
	SaltLength uint32
	// KeyLength is the derived key (hash) size in bytes. Defaults to 32 when zero.
	KeyLength uint32
}

func (c Argon2idConfig) withDefaults() Argon2idConfig {
	if c.Memory == 0 {
		c.Memory = 64 * 1024
	}
	if c.Iterations == 0 {
		c.Iterations = 3
	}
	if c.Parallelism == 0 {
		c.Parallelism = 2
	}
	if c.SaltLength == 0 {
		c.SaltLength = 16
	}
	if c.KeyLength == 0 {
		c.KeyLength = 32
	}
	return c
}

// Argon2idHasher hashes and verifies passwords using Argon2id. Hashes are encoded in
// the standard PHC string format:
//
//	$argon2id$v=19$m=<memory>,t=<iterations>,p=<parallelism>$<salt>$<hash>
//
// so Verify can recover the parameters and salt used by Hash without any extra config.
type Argon2idHasher struct {
	cfg Argon2idConfig
}

// NewArgon2id creates an Argon2idHasher from cfg, defaulting unset fields to
// OWASP-recommended parameters.
func NewArgon2id(cfg Argon2idConfig) *Argon2idHasher {
	return &Argon2idHasher{cfg: cfg.withDefaults()}
}

// Hash returns a PHC-encoded Argon2id hash of password.
func (h *Argon2idHasher) Hash(password string) (string, error) {
	salt := make([]byte, h.cfg.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, h.cfg.Iterations, h.cfg.Memory, h.cfg.Parallelism, h.cfg.KeyLength)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.cfg.Memory, h.cfg.Iterations, h.cfg.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify reports whether password matches hash, a PHC-encoded Argon2id hash previously
// produced by Hash.
func (h *Argon2idHasher) Verify(password, hash string) (bool, error) {
	params, salt, key, err := decodeArgon2idHash(hash)
	if err != nil {
		return false, err
	}

	otherKey := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, uint32(len(key)))

	return subtle.ConstantTimeCompare(key, otherKey) == 1, nil
}

type argon2idParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

// decodeArgon2idHash parses the PHC string format produced by Argon2idHasher.Hash.
func decodeArgon2idHash(hash string) (params argon2idParams, salt, key []byte, err error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return params, nil, nil, errors.New("hasher: invalid argon2id hash format")
	}

	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return params, nil, nil, fmt.Errorf("hasher: invalid argon2id version: %w", err)
	}
	if version != argon2.Version {
		return params, nil, nil, fmt.Errorf("hasher: unsupported argon2id version %d", version)
	}

	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.iterations, &params.parallelism); err != nil {
		return params, nil, nil, fmt.Errorf("hasher: invalid argon2id parameters: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, fmt.Errorf("hasher: invalid argon2id salt: %w", err)
	}

	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params, nil, nil, fmt.Errorf("hasher: invalid argon2id hash: %w", err)
	}

	return params, salt, key, nil
}
