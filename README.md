# go-escpos

[![Go Report Card](https://goreportcard.com/badge/github.com/joeyak/go-escpos)](https://goreportcard.com/report/github.com/joeyak/go-escpos)
![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)
[![GoDoc](https://godoc.org/github.com/joeyak/go-escpos?status.svg)](https://godoc.org/github.com/joeyak/go-escpos)

This is a package for writing to ESC/POS Thermal Printers.

The main printer used in development is the HOIN POS-80-Series Thermal Printer (HOP-E802).

This package has no dependencies. The only dependency in the go.mod is `go-arg` for the `printhis` demo utility.

## Usage

Connect to the printer with an io.ReadWriter and then send commands

```go
package main

import (
	"fmt"
	"net"

	"github.com/joeyak/go-escpos"
)

func main() {

	conn, err := net.Dial("tcp", escpos.DefaultPrinterIP)
	if err != nil {
		fmt.Println("unable to dial:", err)
		return
	}
	defer conn.Close()

	printer := escpos.NewPrinter(conn)

	for i := 0; i < 5; i++ {
		printer.Println("Hello World!")
	}

	printer.FeedLines(5)
	printer.Cut()
}
```

## Demo Utility

The program in `./cmd/printhis/` is a demo utility to demonstrate some basic printing use cases.

## Testing

What? Did I hear you ask for testing? You think we make useless mocks that only tests our assumptions about the hoin printer instead of REAL **HONEST** ***GOOD*** boots on the ground testing.

Run `go run ./cmd/test-printer/` to print out our test program.

Really, how are we supposed to tests without a firmware dump? Total incongruity.

Also the test program assumes some things will work line printing and the such, cause how can we test functions without that. It'd be obvious if nothing prints. The goal is to test all the extra functions like horizontal tabbing, justifications, images, etc.

## Development

View the [docs](./docs/) directory for development progress of commands and programmer manuals
