package main

import (
	"fmt"
	"os"
	"net"
	"io"
	"strings"
	"strconv"
	"path/filepath"
	"image"
	"image/png"
	"image/jpeg"

	"github.com/alexflint/go-arg"
	"github.com/joeyak/hoin-printer"
)

type CmdText struct {
	Input    string `arg:"positional" help:"Print text from a file.  STDIN is used if no filename is given or the filename is a single dash."`
	TabWidth int    `arg:"-t,--tab-width" default:"4" help:"Width of the tabstop in spaces."`
}

type CmdTabs struct {
	TabWidth int `arg:"-t,--tab-width" default:"4" help:"Width of the tabstop in spaces."`
}

type CmdImage struct {
	Input string `arg:"positional,required" help:"Image file to print.  Currently supports PNG and JPEG image formats."`
}

type CmdCut struct { }

type CmdFeed struct {
	Amount int `arg:"positional,required" help:"Amount to feed.  If --lines is used, feed this number of lines.  Otherwise it feeds by units defined by the GS P command."`
	Lines bool `arg:"-l,--lines" help:"Use the line height as the unit of measurement."`
}

type Arguments struct {
	Text  *CmdText  `arg:"subcommand:text"  help:"Print text"`
	Tabs  *CmdTabs  `arg:"subcommand:tabs"  help:"Print the tabstop locations"`
	Image *CmdImage `arg:"subcommand:image" help:"Print an image"`
	Cut   *CmdCut   `arg:"subcommand:cut"   help:"Cut the paper"`
	Feed  *CmdFeed  `arg:"subcommand:feed"  help:"Feed the paper"`

	Address string `arg:"-a,--addr" help:"IP address and port of printer"`
	Device string `arg:"-d,--dev" help:"USB device of printer"`
}

func (a *Arguments) Description() string {
	return `
printhis is a demo utility used to demonstrate some basic printing use cases.
`
}

func main() {
	args := &Arguments{}
	arg.MustParse(args)

	if args.Address == "" && args.Device == "" {
		args.Address = "192.168.1.23:9100"
	}

	printer, closer, err := connect(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closer.Close()

	err = run(args, printer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func connect(args *Arguments) (*hoin.Printer, io.Closer, error) {
	var printer hoin.Printer
	var cl io.Closer
	if args.Address != "" {
		conn, err := net.Dial("tcp", args.Address)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to dial: %w", err)
		}
		cl = conn
		printer = hoin.NewPrinter(conn)

	} else if args.Device != "" {
		file, err := os.OpenFile(args.Device, os.O_RDWR, 0660)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to open device: %w", err)
		}
		cl = file
		printer = hoin.NewPrinter(file)
	} else {
		return nil, nil, fmt.Errorf("Unable to determine printer address")
	}

	return &printer, cl, nil
}

func run(args *Arguments, printer *hoin.Printer) error {
	switch {
	case args.Feed != nil:
		var err error
		if args.Feed.Lines {
			err = printer.FeedLines(args.Feed.Amount)
		} else {
			err = printer.Feed(args.Feed.Amount)
		}

		if err != nil {
			return err
		}

	case args.Cut != nil:
		printer.Cut()

	case args.Text != nil:
		var raw []byte
		var err error
		if args.Text.Input == "" || args.Text.Input == "-" {
			raw, err = io.ReadAll(os.Stdin)
		} else {
			raw, err = os.ReadFile(args.Text.Input)
		}
		if err != nil {
			return err
		}

		words := strings.Split(string(raw), " ")

		for _, word := range words {
			err = printer.Print(word+" ")
			if err != nil {
				return err
			}

			//_, err = printer.TransmitErrorStatus()
			//if err != nil {
			//	return fmt.Errorf("TransmitErrorStatus(): %w", err)
			//}
		}

		//err = printer.Print(string(raw))
		//if err != nil {
		//	return err
		//}

	case args.Tabs != nil:
		var err error
		vals := []string{}

		err = printer.SetTabs(args.Tabs.TabWidth)
		if err != nil {
			return err
		}

		for i := 0; i < 33; i++ {
			vals = append(vals, strconv.Itoa(i))
		}

		err = printer.SetTabs(args.Tabs.TabWidth)
		if err != nil {
			return fmt.Errorf("SetTabs(): %w", err)
		}
		err = printer.Println(strings.Join(vals, "\t"))
		if err != nil {
			return fmt.Errorf("Println(): %w", err)
		}

		_, err = printer.TransmitErrorStatus()
		if err != nil {
			return fmt.Errorf("TransmitErrorStatus(): %w", err)
		}

	case args.Image != nil:
		file, err := os.Open(args.Image.Input)
		if err != nil {
			return err
		}
		defer file.Close()
		var img image.Image

		switch filepath.Ext(args.Image.Input) {
		case ".png":
			img, err = png.Decode(file)

		case ".jpg", ".jpeg":
			img, err = jpeg.Decode(file)

		default:
			return fmt.Errorf("unsupported image format: %s", filepath.Ext(args.Image.Input))
		}

		if err != nil {
			return err
		}

		err = printer.PrintImage24(img, hoin.DoubleDensity)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("Invalid command")
	}

	return nil
}
