// Package hasher provides a generic password hasher with pluggable drivers.
package hasher

import "fmt"

// Driver selects the password hashing algorithm used by New.
type Driver string

const (
	// DriverBcrypt selects bcrypt. This is the default driver.
	DriverBcrypt Driver = "bcrypt"
	// DriverArgon2id selects Argon2id.
	DriverArgon2id Driver = "argon2id"
)

// Config selects and configures a Hasher for New.
type Config struct {
	// Driver selects the hashing algorithm. Defaults to DriverBcrypt when empty.
	Driver Driver
	// Bcrypt configures the bcrypt driver. Only used when Driver is DriverBcrypt.
	Bcrypt BcryptConfig
	// Argon2id configures the argon2id driver. Only used when Driver is DriverArgon2id.
	Argon2id Argon2idConfig
}

// Hasher hashes passwords and verifies a password against a previously produced hash.
type Hasher interface {
	// Hash returns an encoded hash of password, safe to store — it embeds the algorithm's
	// parameters and a random salt, so verifying it later doesn't require any extra config.
	Hash(password string) (string, error)
	// Verify reports whether password matches hash, an encoded hash previously produced by Hash.
	Verify(password, hash string) (bool, error)
}

// New builds a Hasher for the driver selected by cfg. With no Config, it builds a
// BcryptHasher with default settings. It returns an error if Driver is set to an
// unsupported value.
func New(cfg ...Config) (Hasher, error) {
	c := Config{Driver: DriverBcrypt}
	if len(cfg) > 0 {
		c = cfg[0]
		if c.Driver == "" {
			c.Driver = DriverBcrypt
		}
	}

	switch c.Driver {
	case DriverBcrypt:
		return NewBcrypt(c.Bcrypt), nil
	case DriverArgon2id:
		return NewArgon2id(c.Argon2id), nil
	default:
		return nil, fmt.Errorf("hasher: unsupported driver %q", c.Driver)
	}
}
