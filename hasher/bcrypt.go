package hasher

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// BcryptConfig configures a BcryptHasher.
type BcryptConfig struct {
	// Cost is the bcrypt work factor, between bcrypt.MinCost (4) and bcrypt.MaxCost (31).
	// Defaults to bcrypt.DefaultCost (10) when zero.
	Cost int
}

// BcryptHasher hashes and verifies passwords using bcrypt. The cost, version, and a
// random salt are embedded in every hash it produces, so Verify needs no extra config.
type BcryptHasher struct {
	cost int
}

// NewBcrypt creates a BcryptHasher from cfg, defaulting Cost to bcrypt.DefaultCost when unset.
func NewBcrypt(cfg BcryptConfig) *BcryptHasher {
	cost := cfg.Cost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash returns a bcrypt hash of password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Verify reports whether password matches hash, a bcrypt hash previously produced by Hash.
func (h *BcryptHasher) Verify(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, err
	}
}
