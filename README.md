# go-kit

Small set of Go utility packages for building HTTP APIs.

## Install

```bash
go get github.com/akfaiz/go-kit
```

## Packages

| Package | Description |
| --- | --- |
| [`hasher`](./hasher) | Generic password hasher (bcrypt, argon2id) |
| [`problem`](./problem) | RFC 7807 structured error responses |
| [`validator`](./validator) | i18n-aware struct validation |

## Development

```bash
make build
make lint
make test
```
