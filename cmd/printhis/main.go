package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/joeyak/go-escpos"
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

type CmdCut struct{}

type CmdFeed struct {
	Amount int  `arg:"positional,required" help:"Amount to feed.  If --lines is used, feed this number of lines.  Otherwise it feeds by units defined by the GS P command."`
	Lines  bool `arg:"-l,--lines" help:"Use the line height as the unit of measurement."`
}

type CmdSizes struct {
	Text string `arg:"positional" help:"Sample text to use instead of 'size WxH'."`
}

type Arguments struct {
	Text  *CmdText  `arg:"subcommand:text"  help:"Print text"`
	Tabs  *CmdTabs  `arg:"subcommand:tabs"  help:"Print the tabstop locations"`
	Image *CmdImage `arg:"subcommand:image" help:"Print an image"`
	Cut   *CmdCut   `arg:"subcommand:cut"   help:"Cut the paper"`
	Feed  *CmdFeed  `arg:"subcommand:feed"  help:"Feed the paper"`
	Sizes *CmdSizes `arg:"subcommand:sizes" help:"Print out all character width and height combinations"`

	Address string `arg:"-a,--addr" help:"IP address and port of printer"`
	Device  string `arg:"-d,--dev" help:"USB device of printer"`
	Justify string `arg:"-j,--justify"`

	CharWidth  int `arg:"--char-width"  help:"Character width. Valid values are 0-7." default:"-1"`
	CharHeight int `arg:"--char-height" help:"Character width. Valid values are 0-7." default:"-1"`

	UpsideDown string `arg:"--upside-down"`

	EnvDevice string `arg:"env:ESCPOS_DEVICE"`
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
		if args.EnvDevice != "" {
			err := getEnvDevice(args)
			if err != nil {
				args.Address = escpos.DefaultHoinIP
				fmt.Println(err)
			}
		} else {
			args.Address = escpos.DefaultHoinIP
		}
	}

	printer, err := connect(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer printer.Close()

	err = justify(args, printer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if args.CharWidth > -1 || args.CharHeight > -1 {
		err = charSize(args, printer)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if args.UpsideDown != "" {
		switch args.UpsideDown {
		case "on", "true":
			err = printer.SetUpsideDown(true)
		default:
			err = printer.SetUpsideDown(false)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	err = run(args, printer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getEnvDevice(args *Arguments) error {
	if args.EnvDevice == "" {
		return nil
	} else if !strings.Contains(args.EnvDevice, `://`) {
		err := fmt.Errorf("invalid ESCPOS_DEVICE value %q\n", args.EnvDevice)
		args.EnvDevice = ""
		return err
	}

	parts := strings.SplitN(args.EnvDevice, `://`, 2)
	if len(parts) != 2 {
		err := fmt.Errorf("invalid ESCPOS_DEVICE value %q\n", args.EnvDevice)
		args.EnvDevice = ""
		return err
	}

	switch parts[0] {
	case "file":
		args.Device = parts[1]
	case "tcp":
		args.Address = parts[1]
	default:
		err := fmt.Errorf("invalid ESCPOS_DEVICE value %q\n", args.EnvDevice)
		args.EnvDevice = ""
		return err
	}

	return nil
}

func charSize(args *Arguments, printer *escpos.Printer) error {
	if args.CharWidth < 0 {
		args.CharWidth = 0
	}

	if args.CharHeight < 0 {
		args.CharHeight = 0
	}

	return printer.SetCharacterSize(args.CharWidth, args.CharHeight)
}

func justify(args *Arguments, printer *escpos.Printer) error {
	if args.Justify == "" {
		return nil
	}

	var j escpos.Justification
	switch strings.ToLower(args.Justify) {
	case "left":
		j = escpos.LeftJustify
	case "right":
		j = escpos.RightJustify
	case "center":
		j = escpos.CenterJustify
	default:
		return fmt.Errorf("invalid justification")
	}

	err := printer.Justify(j)
	if err != nil {
		return err
	}

	//_, err = printer.TransmitErrorStatus()
	//if err != nil {
	//	return fmt.Errorf("TransmitErrorStatus(): %w", err)
	//}

	return nil
}

func connect(args *Arguments) (*escpos.Printer, error) {
	if args.Address != "" {
		printer, err := escpos.NewIpPrinter(args.Address)
		if err != nil {
			return nil, err
		}
		return &printer, nil
	} else if args.Device != "" {
		file, err := os.OpenFile(args.Device, os.O_RDWR, 0660)
		if err != nil {
			return nil, fmt.Errorf("unable to open device: %w", err)
		}
		printer := escpos.NewPrinter(file)
		return &printer, nil
	}
	return nil, fmt.Errorf("unable to determine printer address")
}

func run(args *Arguments, printer *escpos.Printer) error {
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
		err := printer.Cut()
		if err != nil {
			return err
		}

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

		words := strings.Split(strings.TrimRight(string(raw), "\r\n"), " ")

		for _, word := range words {
			err = printer.Print(word + " ")
			if err != nil {
				return err
			}

			//_, err = printer.TransmitErrorStatus()
			//if err != nil {
			//	return fmt.Errorf("TransmitErrorStatus(): %w", err)
			//}
		}

		err = printer.LF()
		//err = printer.Print(string(raw))
		if err != nil {
			return err
		}

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

		err = printer.PrintImage24(img, escpos.DoubleDensity)
		if err != nil {
			return err
		}

	case args.Sizes != nil:
		for w := 0; w < 8; w++ {
			for h := 0; h < 8; h++ {
				err := printer.SetCharacterSize(w, h)
				if err != nil {
					return err
				}

				if args.Sizes.Text != "" {
					err = printer.Println(args.Sizes.Text)
				} else {
					err = printer.Printf("size %dx%d", w, h)
					if err != nil {
						return err
					}

					err = printer.LF()
				}
				if err != nil {
					return err
				}
			}
		}

	default:
		return fmt.Errorf("Invalid command")
	}

	return nil
}
