package main

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"slices"

	"github.com/alexflint/go-arg"
	"github.com/joeyak/go-escpos"
	"github.com/joeyak/go-escpos/cmd"
)

func connect(addresses []string) (escpos.Printer, error) {
	if len(addresses) == 0 {
		return escpos.Printer{}, fmt.Errorf("unable to determine printer address")
	}

	var printers []escpos.Printer

	for _, address := range addresses {
		if _, err := os.Open(address); err == nil {
			file, err := os.OpenFile(address, os.O_RDWR, 0660)
			if err != nil {
				return escpos.Printer{}, fmt.Errorf("unable to open device: %w", err)
			}
			printers = append(printers, escpos.NewPrinter(file))
			continue
		}

		printer, err := escpos.NewIpPrinter(address)
		if err != nil {
			return escpos.Printer{}, err
		}
		printers = append(printers, printer)
	}

	if len(printers) == 1 {
		return printers[0], nil
	}

	return escpos.NewPrinter(cmd.NewMultiPrinter(printers...)), nil
}

func runTest(addresses []string, testName string, testFunc func(escpos.Printer) error) error {
	printer, err := connect(addresses)
	if err != nil {
		return fmt.Errorf("failed test %s: %w", testName, err)
	}
	defer printer.Close()

	printer.Initialize()
	printer.Println("=== ", testName, " ===")
	defer printer.LF()

	err = testFunc(printer)
	if err != nil {
		return fmt.Errorf("failed test %s: %w", testName, err)
	}

	return nil
}

func cleanup(addresses []string) {
	printer, err := connect(addresses)
	if err != nil {
		fmt.Printf("could not create new printer to feed lines: %s\n", err)
		os.Exit(1)
	}
	defer printer.Close()

	printer.Println("##### ", time.Now().Format(time.DateOnly+" "+time.Kitchen), " #####")
	printer.FeedLines(10)
}

type Arguments struct {
	Addresses []string `arg:"positional" help:"IP address and port of printer or USB device"`
	Filters   []string `arg:"-f,--filter" help:"the name of the function to test - test<FILTER>"`
	List      bool     `arg:"--list" help:"print out list of test functions"`
}

func main() {
	args := &Arguments{}
	arg.MustParse(args)

	if len(args.Addresses) == 0 {
		args.Addresses = []string{escpos.DefaultHoinIP}
	}

	tests := []func(escpos.Printer) error{
		testBeep,
		testHT,
		testLineSpacing,
		testBold,
		testRotate90,
		testReversePrinter,
		testFonts,
		testJustify,
		testFeed,
		testFeedLines,
	}

	if args.List {
		fmt.Println("Test Filters:")
		for _, test := range tests {
			fmt.Println(strings.TrimPrefix(runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name(), "main.test"))
		}
		return
	}

	var errors []error

	for i, test := range tests {
		testName := strings.TrimPrefix(runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name(), "main.test")

		if len(args.Filters) > 0 && !slices.Contains(args.Filters, testName) {
			continue
		}

		fmt.Printf("Running test [%d/%d] %s - ", i+1, len(tests), testName)

		err := runTest(args.Addresses, testName, test)
		if err != nil {
			fmt.Println("fail")
			errors = append(errors, err)
		}

		fmt.Println("pass")

		time.Sleep(time.Millisecond * 250)
	}

	cleanup(args.Addresses)

	if len(errors) > 0 {
		fmt.Printf("%d errors occured\n", len(errors))
		for _, err := range errors {
			fmt.Println(err)
		}
		os.Exit(1)
	}

}

func testBeep(printer escpos.Printer) error {
	return printer.Beep(1, 1)
}

func testHT(printer escpos.Printer) error {
	err := printer.Print("-")
	if err != nil {
		return fmt.Errorf("could not print HT prefix: %w", err)
	}

	err = printer.SetHT(10)
	if err != nil {
		return fmt.Errorf("could not set HT positions: %w", err)
	}
	defer printer.SetHT()

	err = printer.HT()
	if err != nil {
		return fmt.Errorf("could not print HT prefix: %w", err)
	}

	err = printer.Println("- 10 character tab")
	if err != nil {
		return fmt.Errorf("could not print HT suffix: %w", err)
	}

	err = printer.Println("~", strings.Repeat("-", 9), "~")
	if err != nil {
		return fmt.Errorf("could not print ruler line: %w", err)
	}

	return nil
}

func testLineSpacing(printer escpos.Printer) error {
	defer printer.ResetLineSpacing()

	for _, spacing := range []int{0, 255} {
		err := printer.SetLineSpacing(spacing)
		if err != nil {
			return fmt.Errorf("could not set line spacing to %d: %w", spacing, err)
		}

		err = printer.Printf("Spacing %d start\n", spacing)
		if err != nil {
			return fmt.Errorf("could not print line spacing %d start", spacing)
		}
		err = printer.Printf("Spacing %d end\n", spacing)
		if err != nil {
			return fmt.Errorf("could not print line spacing %d end", spacing)
		}
	}

	err := printer.ResetLineSpacing()
	if err != nil {
		return err
	}

	err = printer.Println("Reset spacing start")
	if err != nil {
		return fmt.Errorf("could not print line spacing reset start: %w", err)
	}

	err = printer.Println("Reset spacing end")
	if err != nil {
		return fmt.Errorf("could not print line spacing reset end: %w", err)
	}

	return nil
}

func testBold(printer escpos.Printer) error {
	defer printer.SetBold(false)

	err := printer.Print("Normal ")
	if err != nil {
		return fmt.Errorf("could not print start control text: %w", err)
	}

	err = printer.SetBold(true)
	if err != nil {
		return err
	}

	err = printer.Print("Bold")
	if err != nil {
		return fmt.Errorf("could not print bold text: %w", err)
	}

	err = printer.SetBold(false)
	if err != nil {
		return err
	}

	err = printer.Println(" Normal")
	if err != nil {
		return fmt.Errorf("could not print end control text: %w", err)
	}

	return nil
}

func testRotate90(printer escpos.Printer) error {
	defer printer.SetRotate90(false)

	err := printer.Println("Control Text")
	if err != nil {
		return fmt.Errorf("could not print control text: %w", err)
	}

	err = printer.SetRotate90(true)
	if err != nil {
		return err
	}

	err = printer.Println("Rotated Text")
	if err != nil {
		return fmt.Errorf("could not print rotated text: %w", err)
	}

	err = printer.SetRotate90(false)
	if err != nil {
		return err
	}

	return nil
}

func testReversePrinter(printer escpos.Printer) error {
	defer printer.SetReversePrinting(false)

	err := printer.Println("Control Text")
	if err != nil {
		return fmt.Errorf("could not print control text: %w", err)
	}

	err = printer.SetReversePrinting(true)
	if err != nil {
		return err
	}

	err = printer.Println("Reversed Text")
	if err != nil {
		return fmt.Errorf("could not print reversed text: %w", err)
	}

	return nil
}

func testFonts(printer escpos.Printer) error {
	defer printer.SetFont(escpos.FontA)

	err := printer.SetFont(escpos.FontA)
	if err != nil {
		return err
	}

	err = printer.Println("Font A")
	if err != nil {
		return fmt.Errorf("could not print Font A: %w", err)
	}

	err = printer.SetFont(escpos.FontB)
	if err != nil {
		return err
	}

	err = printer.Println("Font B")
	if err != nil {
		return fmt.Errorf("could not print Font B: %w", err)
	}

	return nil
}

func testJustify(printer escpos.Printer) error {
	defer printer.Justify(escpos.LeftJustify)

	err := printer.Justify(escpos.LeftJustify)
	if err != nil {
		return err
	}

	err = printer.Println("Left Justify")
	if err != nil {
		return fmt.Errorf("could not print Left Justify: %w", err)
	}

	err = printer.Justify(escpos.CenterJustify)
	if err != nil {
		return err
	}

	err = printer.Println("Center Justify")
	if err != nil {
		return fmt.Errorf("could not print Center Justify: %w", err)
	}

	err = printer.Justify(escpos.RightJustify)
	if err != nil {
		return err
	}

	err = printer.Println("Right Justify")
	if err != nil {
		return fmt.Errorf("could not print Right Justify: %w", err)
	}

	return nil
}

func testFeed(printer escpos.Printer) error {
	err := printer.Println("------------")
	if err != nil {
		return fmt.Errorf("could not print before line: %w", err)
	}

	for _, lines := range []int{10, 100, 255} {
		err = printer.Feed(lines)
		if err != nil {
			return err
		}

		err = printer.Printf("------------ %d\n", lines)
		if err != nil {
			return fmt.Errorf("could not print after line for %d lines: %w", lines, err)
		}
	}

	return nil
}

func testFeedLines(printer escpos.Printer) error {
	err := printer.Println("------------")
	if err != nil {
		return fmt.Errorf("could not print before line: %w", err)
	}

	for _, lines := range []int{1, 5, 10} {
		err = printer.FeedLines(lines)
		if err != nil {
			return err
		}

		err = printer.Printf("------------ %d\n", lines)
		if err != nil {
			return fmt.Errorf("could not print after line for %d lines: %w", lines, err)
		}
	}

	return nil
}
