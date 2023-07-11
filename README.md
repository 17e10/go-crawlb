# go-crawlb

[![GoDev][godev-image]][godev-url]

go-crawl パッケージは Web サイトの巡回ユーティリティを提供します.

## Usage

```go
import "github.com/17e10/go-crawlb"

cl, err := crawlb.NewClient(ctx, 2 * time.Second, cacheDir, numTx)
cl.NewTransaction()
resp, err := cl.Get("https://google.com/")
defer resp.Body.Close()
```

## License

This software is released under the MIT License, see LICENSE.

## Author

17e10

[godev-image]: https://pkg.go.dev/badge/github.com/17e10/go-crawlb
[godev-url]: https://pkg.go.dev/github.com/17e10/go-crawlb
