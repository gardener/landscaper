# buflogr

[![Go Reference](https://pkg.go.dev/badge/github.com/tonglil/buflogr.svg)](https://pkg.go.dev/github.com/tonglil/buflogr)
<!-- ![test](https://github.com/tonglil/buflogr/workflows/test/badge.svg) -->
[![Go Report Card](https://goreportcard.com/badge/github.com/tonglil/buflogr)](https://goreportcard.com/report/github.com/tonglil/buflogr)

A [logr](https://github.com/go-logr/logr) LogSink implementation using [bytes.Buffer](https://pkg.go.dev/bytes).

## Usage

```go
import (
	"bytes"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/tonglil/buflogr"
)

func main() {
	var buf bytes.Buffer
	var log logr.Logger = buflogr.NewWithBuffer(&buf)

	log = log.WithName("my app")
	log = log.WithValues("format", "none")

	log.Info("Logr in action!", "the answer", 42)

	fmt.Print(buf.String())
}
```

## Implementation Details

This is a simple log adapter to log messages into a buffer.
Useful for testing.
