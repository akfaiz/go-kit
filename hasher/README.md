# hasher

Generic password hasher with pluggable drivers. Supported drivers: `bcrypt` (default)
and `argon2id`.

## Usage

```go
import "github.com/akfaiz/go-kit/hasher"

h, err := hasher.New() // defaults to bcrypt

hash, err := h.Hash("s3cr3t-password")

ok, err := h.Verify("s3cr3t-password", hash)
```

## Choosing a driver

```go
h, err := hasher.New(hasher.Config{
	Driver: hasher.DriverArgon2id,
	Argon2id: hasher.Argon2idConfig{
		Memory:      64 * 1024, // KiB
		Iterations:  3,
		Parallelism: 2,
	},
})
```

`New` returns an error for an unset-but-invalid or unsupported `Driver` value.

### bcrypt

```go
h, err := hasher.New(hasher.Config{
	Driver: hasher.DriverBcrypt,
	Bcrypt: hasher.BcryptConfig{Cost: 12}, // defaults to bcrypt.DefaultCost (10)
})
```

### argon2id

Hashes are encoded using the standard PHC string format
(`$argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>`), so `Verify` recovers the parameters
and salt used by `Hash` without any extra config.

```go
h, err := hasher.New(hasher.Config{
	Driver: hasher.DriverArgon2id,
	Argon2id: hasher.Argon2idConfig{
		Memory:      64 * 1024, // KiB, defaults to 64 MiB
		Iterations:  3,          // defaults to 3
		Parallelism: 2,          // defaults to 2
		SaltLength:  16,         // defaults to 16 bytes
		KeyLength:   32,         // defaults to 32 bytes
	},
})
```

## API

| Type / Func | Description |
| --- | --- |
| `Hasher` | Interface with `Hash(password string) (string, error)` and `Verify(password, hash string) (bool, error)` |
| `New(cfg ...Config) (Hasher, error)` | Build a `Hasher` for `cfg.Driver` (defaults to `DriverBcrypt`) |
| `NewBcrypt(BcryptConfig) *BcryptHasher` | Build a bcrypt `Hasher` directly |
| `NewArgon2id(Argon2idConfig) *Argon2idHasher` | Build an argon2id `Hasher` directly |
