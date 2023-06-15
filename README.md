# aios

Abstract IO Streams(fork from github.com/cli/cli)

## How to use

```go
// in production
import "github.com/alswl/aios"

ios := aios.System()

// in testing
import "github.com/alswl/aios"

ios, stdin, stdout, stderr := aios.Test()
```
