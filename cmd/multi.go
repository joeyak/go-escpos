package cmd

import (
	"github.com/joeyak/go-escpos"
)

type MultiPrinter struct {
	dst []escpos.Printer
}

func NewMultiPrinter(printers ...escpos.Printer) MultiPrinter {
	return MultiPrinter{dst: printers}
}

func (mp MultiPrinter) Read(p []byte) (n int, err error) {
	for _, printer := range mp.dst {
		n, err := printer.Read(p)
		if err != nil {
			return n, err
		}
	}
	return 0, nil
}

func (mp MultiPrinter) Write(p []byte) (n int, err error) {
	for _, printer := range mp.dst {
		n, err := printer.Write(p)
		if err != nil {
			return n, err
		}
	}
	return 0, nil
}

func (mp MultiPrinter) Close() error {
	for _, printer := range mp.dst {
		err := printer.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
