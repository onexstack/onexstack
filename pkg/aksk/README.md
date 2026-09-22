# Access Key / Secret Key (pkg/aksk)

[![Go Version](https://img.shields.io/github/go-mod/go-version/onexstack/onexstack)](https://github.com/onexstack/onexstack)
[![Go Report Card](https://goreportcard.com/badge/github.com/onexstack/onexstack)](https://goreportcard.com/report/github.com/onexstack/onexstack)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## Summary

`pkg/aksk` is the unified Access Key / Secret Key package for OneXStack. It
mints the long-lived credential pair a client uses for programmatic access to a
OneX service, and provides the one-way function a store keeps instead of the
secret itself.

## Specification

| Half        | Format                | Length | Charset                  | Entropy |
| ----------- | --------------------- | ------ | ------------------------ | ------- |
| Access key  | `ak-` + 20 characters | 23     | Base62 (`A-Za-z0-9`)     | ~119 bit |
| Secret key  | `sk-` + 32 characters | 35     | Base62 (`A-Za-z0-9`)     | ~190 bit |

Both halves come from `crypto/rand`. The Base62 alphabet excludes `-` and `_`,
so a credential splits into exactly two parts on the prefix separator and a body
character can never be mistaken for it.

Bytes are drawn by rejection sampling rather than reduced modulo 62, which would
make the first 8 characters of the alphabet ~1.6% more likely than the rest.

## Demo

```go
import (
    "fmt"

    "github.com/onexstack/onexstack/pkg/aksk"
)

func main() {
    accessKey, secretKey, err := aksk.Generate()
    if err != nil {
        panic(err)
    }

    fmt.Println(accessKey)                 // ak-7fK2mQ9xLp4RtZ8wNc3B
    fmt.Println(secretKey)                 // sk-A9dLm2Xq7Pv4TzR8wNc3Bk6Hj5Ys1Ue0
    fmt.Println(aksk.Digest(secretKey))    // sha256 hex, what a store keeps
}
```

## Getting Started

```bash
go get -u github.com/onexstack/onexstack/pkg/aksk
```

## API

```go
func Generate() (accessKey, secretKey string, err error)
func GenerateAccessKey() (string, error)
func GenerateSecretKey() (string, error)
func Digest(secretKey string) string
```

## Handling the secret key

`Generate` returns the secret key once, and nothing in this package can recover
it from `Digest`. A service should hand the secret to the caller in the creation
response and store only the digest — the trade `iam.aksk_keys` makes.

## Contributing

Contributions are welcome. Please open an issue or submit a pull request.

## License

MIT
