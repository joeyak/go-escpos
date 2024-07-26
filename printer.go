package escpos

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"net"
	"strings"
)

const (
	// Default ip and port for hoin printers
	DefaultHoinIP = "192.168.1.23:9100"

	HT  = 0x09
	LF  = 0x0A
	CR  = 0x0D
	GS  = 0x1D
	ESC = 0x1B
	DLE = 0x10
)

type Font int

const (
	FontA Font = iota
	FontB
)

type Justification int

const (
	LeftJustify Justification = iota
	CenterJustify
	RightJustify
)

type HRIPosition int

const (
	HRINone HRIPosition = iota
	HRIAbove
	HRIBelow
	HRIBoth
)

// Density represents the DPI to use when printing images.
type Density int

const (
	// SingleDensity is 90dpi
	SingleDensity Density = iota
	// DoubleDensity is 180dpi
	DoubleDensity
)

type BarCode int

const (
	BcUPCA BarCode = iota
	BcUPCE
	BcJAN13
	BcJAN8
	BcCODE39
	BcITF
	BcCODABAR
	BcCODE93  BarCode = 72
	BcCODE123 BarCode = 73
)

type PrintModeMask byte

const (
	// bit 0: Font selection, 0: Font A (normal), 1: Font B (thin)
	ThinFont PrintModeMask = 1 << 0
	// bit 1 and 2 are unused
	// bit 3: Bold, 0: Off, 1: On
	Bold PrintModeMask = 1 << 3
	// bit 4: Double height, 0: Off, 1: On
	DoubleHeight PrintModeMask = 1 << 4
	// bit 5: Double width, 0: Off, 1
	DoubleWidth PrintModeMask = 1 << 5
	// bit 6 unused
	// bit 7: Underline, 0: Off, 1: On
	Underline PrintModeMask = 1 << 7
)

var (
	lengthBarcodes = []BarCode{BcCODE93, BcCODE123}
	allBarcodes    = append(lengthBarcodes, BcUPCA, BcUPCE, BcJAN13, BcJAN8, BcCODE39, BcITF, BcCODABAR)
)

func inSlice[T ~int](v T, s ...T) bool {
	for _, a := range s {
		if v == a {
			return true
		}
	}
	return false
}

func checkEnum[T ~int](e T, enums ...T) error {
	if inSlice(e, enums...) {
		return nil
	}
	return fmt.Errorf("%v was not a valid choice from %v", e, enums)
}

func checkRange(n, min, max int, info string) error {
	if n < min || max < n {
		return fmt.Errorf("%s must be between %d and %d", info, min, max)
	}
	return nil
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

type Printer struct {
	dst io.ReadWriteCloser
}

func NewPrinter(dst io.ReadWriteCloser) Printer {
	return Printer{
		dst: dst,
	}
}

func NewIpPrinter(addr string) (Printer, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return Printer{}, fmt.Errorf("unable to dial: %w", err)
	}
	return NewPrinter(conn), nil
}

func (p Printer) Close() error {
	closer, ok := p.dst.(io.Closer)
	if p.dst == nil || !ok {
		return nil
	}

	err := closer.Close()
	if err != nil {
		return fmt.Errorf("could not close printer: %w", err)
	}
	return nil
}

func (p Printer) Write(b []byte) (int, error) {
	n, err := p.dst.Write(b)
	if err != nil {
		return n, fmt.Errorf("could not write to printer: %w", err)
	}
	return n, nil
}

func (p Printer) Read(b []byte) (int, error) {
	n, err := p.dst.Read(b)
	if err != nil {
		return n, fmt.Errorf("could not read from printer: %w", err)
	}
	return n, nil
}

func (p Printer) Initialize() error {
	_, err := p.Write([]byte{ESC, '@'})
	if err != nil {
		return fmt.Errorf("could not initialize printer: %w", err)
	}
	return nil
}

// Beep makes a beep sound n times for t duration
//
// Duration is dependent on the model. For the HOP-E802
// each duration is around 100ms
func (p Printer) Beep(n, t int) error {
	errMsg := "could not beep the printer: %w"

	err := checkRange(n, 1, 9, "n")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	err = checkRange(t, 1, 9, "t")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{ESC, 'B', byte(n), byte(t)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return nil
}

func (p Printer) Print(a ...any) error {
	_, err := p.Write([]byte(fmt.Sprint(a...)))
	if err != nil {
		return fmt.Errorf("could not print %q: %w", a, err)
	}
	return nil
}

func (p Printer) Println(a ...any) error {
	return p.Print(fmt.Sprint(a...) + string(rune(LF)))
}

func (p Printer) Printf(format string, a ...any) error {
	return p.Print(fmt.Sprintf(format, a...))
}

// HT moves the print position to the next horizontal tab position
//
// By default HT will do nothing if SetHT is not called with tab positions
func (p Printer) HT() error {
	_, err := p.Write([]byte{HT})
	if err != nil {
		return fmt.Errorf("could not send HT: %w", err)
	}
	return nil
}

// LF prints the data in the print buffer and feeds one line
func (p Printer) LF() error {
	_, err := p.Write([]byte{LF})
	if err != nil {
		return fmt.Errorf("could not send LF: %w", err)
	}
	return nil
}

// CR prints and does a carriage return
func (p Printer) CR() error {
	_, err := p.Write([]byte{CR})
	if err != nil {
		return fmt.Errorf("could not send CR: %w", err)
	}
	return nil
}

// Cut cuts the paper
func (p Printer) Cut() error {
	_, err := p.Write([]byte{GS, 'V', 0})
	if err != nil {
		return fmt.Errorf("could not cut paper: %w", err)
	}
	return nil
}

// CutFeed feeds the paper n units and then cuts it
func (p Printer) CutFeed(n int) error {
	errMsg := "could not feed and cut the paper: %w"

	err := checkRange(n, 0, 255, "n")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{GS, 'V', 66, byte(n)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

// ResetLineSpacing sets the spacing to the default which
// is 1/6-inch lines (approx. 4.23mm)
func (p Printer) ResetLineSpacing() error {
	_, err := p.Write([]byte{ESC, '2'})
	if err != nil {
		return fmt.Errorf("could not reset line spacing: %w", err)
	}
	return nil
}

// SetLineSpacing sets the line spacing to n * v/h motion units in inches
func (p Printer) SetLineSpacing(n int) error {
	errMsg := "could not set line spacing: %w"

	err := checkRange(n, 0, 255, "n")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{ESC, '3', byte(n)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return err
}

// Feed feeds the paper n units
func (p Printer) Feed(n int) error {
	errMsg := "could not feed paper: %w"

	err := checkRange(n, 0, 255, "n")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{ESC, 'J', byte(n)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return err
}

// FeedLines feeds the paper n lines
func (p Printer) FeedLines(n int) error {
	errMsg := "could not feed lines: %w"

	err := checkRange(n, 0, 255, "n")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{ESC, 'd', byte(n)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return err
}

// SetHT sets the horizontal tab positions
//
// This command cancels previous SetHT commands
// Multiple positions can be set for tabbing
// A max of 32 positions can be set
// Calling SetHT with no argments resets the tab positions
func (p Printer) SetHT(positions ...int) error {
	errMsg := "could not set horizontal tab positions: %w"

	if len(positions) > 32 {
		return fmt.Errorf("more than 32 positions was set")
	}

	var data []byte
	for i, pos := range positions {
		err := checkRange(pos, 1, 255, fmt.Sprintf("position %d", i))
		if err != nil {
			return fmt.Errorf(errMsg, err)
		}

		data = append(data, byte(pos))
	}

	data = append([]byte{ESC, 'D'}, data...)
	data = append(data, 0)

	_, err := p.Write(data)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return nil
}

// SetTabs will set up to 32 tab positions at the given width intervals.  If
// the tab value exceeds 256, fewer than 32 positions will be set.
func (p Printer) SetTabs(width int) error {
	tabs := []int{}
	for i := width; i < 256 && len(tabs) < 32; i += width {
		tabs = append(tabs, i)
	}
	return p.SetHT(tabs...)
}

// SetBold turns emphasized mode on or off
func (p Printer) SetBold(b bool) error {
	_, err := p.Write([]byte{ESC, 'E', boolToByte(b)})
	if err != nil {
		return fmt.Errorf("could not set bold to %t: %w", b, err)
	}
	return nil
}

// SetRotate90 turns on 90 clockwise rotation mode for the text
//
// When text is double-width or double-height the text will be mirrored
func (p Printer) SetRotate90(b bool) error {
	_, err := p.Write([]byte{ESC, 'V', boolToByte(b)})
	if err != nil {
		return fmt.Errorf("could not set bold to %t: %w", b, err)
	}
	return nil
}

// SetReversePrinting sets the white/black printing mode
//
// If b is true then it will print black text on white background
// If b is false then it will print white text on black background
func (p Printer) SetReversePrinting(b bool) error {
	_, err := p.Write([]byte{GS, 'B', boolToByte(b)})
	if err != nil {
		return fmt.Errorf("could not set reverse printing mode: %w", err)
	}
	return nil
}

// SetFont changes the font
//
// n=0 selects font A
// n=1 selects font B
func (p Printer) SetFont(f Font) error {
	errMsg := "could not set font to %v: %w"

	err := checkEnum(f, FontA, FontB)
	if err != nil {
		return fmt.Errorf(errMsg, f, err)
	}

	_, err = p.Write([]byte{ESC, 'M', byte(f)})
	if err != nil {
		return fmt.Errorf(errMsg, f, err)
	}

	return nil
}

// Justify sets the alignment to n
func (p Printer) Justify(j Justification) error {
	errMsg := "could not set justify to %v: %w"

	err := checkEnum(j, CenterJustify, LeftJustify, RightJustify)
	if err != nil {
		return fmt.Errorf(errMsg, j, err)
	}

	_, err = p.Write([]byte{ESC, 'a', byte(j)})
	if err != nil {
		return fmt.Errorf(errMsg, j, err)
	}
	return nil
}

// PrintImage8 prints an image in the 8-bit row format.  In this format each
// row is 8 dots tall.
//
// The density selects the horizontal DPI of the image.  SingleDensity is
// 90dpi while DoubleDensity is 180dpi.  Vertical DPI is always 60dpi for
// 8-bit image data.
//
// No black and white conversion is performed on the provided image.  The
// image should be converted before calling this function.
func (p Printer) PrintImage8(img image.Image, density Density) error {
	imgRect := img.Bounds()
	var err error
	errMsg := "could not print 8 dot image: %w"

	err = checkEnum(density, SingleDensity, DoubleDensity)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	// 8 dot density (meta row is 8 dots tall)
	for y := 0; y < imgRect.Max.Y; y += 8 {
		row := []byte{}
		for x := 0; x < imgRect.Max.X; x++ {
			col := byte(0)

			for i := 0; i < 8; i++ {
				col <<= 1
				// Pad the bottom row to be white
				if y+i >= imgRect.Max.Y {
					continue
				}
				c := color.GrayModel.Convert(img.At(x, y+i)).(color.Gray)
				if c.Y < 0x80 {
					col |= 1
				}
			}

			row = append(row, col)
		}

		data := []byte{ESC, '*', byte(density), byte(len(row)), byte(len(row) >> 8)}

		if err = p.SetLineSpacing(0); err != nil {
			return fmt.Errorf(errMsg, err)
		}
		if _, err = p.Write(append(data, row...)); err != nil {
			return fmt.Errorf(errMsg, err)
		}
		if err = p.LF(); err != nil {
			return fmt.Errorf(errMsg, err)
		}

		// Wait for line to finish
		_, err = p.TransmitErrorStatus()
		if err != nil {
			return fmt.Errorf(errMsg, err)
		}
	}

	return nil
}

// PrintImage24 prints an image in the 24-bit row format.  In this format each
// row is 24 dots tall.
//
// This works the same as PrintImage8() with the only difference being the DPI
// of the printed image.  SingleDensity is 90dpi while DoubleDensity is
// 180dpi.  Vertical DPI is always 180dpi for 24-bit image data.
func (p Printer) PrintImage24(img image.Image, density Density) error {
	imgRect := img.Bounds()
	var err error
	errMsg := "could not print 24 dot image: %w"

	err = checkEnum(density, SingleDensity, DoubleDensity)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	command := []byte{ESC, 0x2A, byte(density + 32), byte(imgRect.Max.X), byte(imgRect.Max.X >> 8)}

	// 24 dot density (meta row is 24 dots tall (3 bytes))
	for y := 0; y < imgRect.Max.Y; y += 24 {
		row := []byte{}

		for x := 0; x < imgRect.Max.X; x++ {
			for z := 0; z < 3; z++ {
				col := byte(0)
				for i := 0; i < 8; i++ {
					col <<= 1
					// Pad the bottom row to be white
					if (y+z*8)+i >= imgRect.Max.Y {
						continue
					}

					c := color.GrayModel.Convert(img.At(x, (y+z*8)+i)).(color.Gray)
					if c.Y < 0x80 {
						col |= 1
					}
				}
				row = append(row, col)
			}
		}

		err = p.SetLineSpacing(0)
		if err != nil {
			return fmt.Errorf(errMsg, err)
		}

		_, err = p.Write(append(command, row...))
		if err != nil {
			return fmt.Errorf(errMsg, err)
		}

		err = p.LF()
		if err != nil {
			return fmt.Errorf(errMsg, err)
		}

		// If data is sent to fast it won't make it to the printer and will
		// stop printing part of the way through an image.  This will also
		// lose any commands sent after the image.  Sleeping for 35ms seems to
		// be the best balance between not printing and reducing banding.
		// time.Sleep(time.Millisecond * 35)

		// Leaving above there just in case below is a footgun, but below should
		// wait till print buffer is done to write more lines
		_, err = p.TransmitErrorStatus()
		if err != nil {
			return fmt.Errorf(errMsg, err)
		}
	}

	return nil
}

// SetHRIPosition sets the printing position of the HRI characters
// in relation to the barcode
func (p Printer) SetHRIPosition(hp HRIPosition) error {
	errMsg := "could not set HRI position: %w"

	err := checkEnum(hp, HRINone, HRIAbove, HRIBelow, HRIBoth)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{GS, 'H', byte(hp)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

// ResetBarCodeHeight sets the bar code height to 162
func (p Printer) ResetBarCodeHeight() error {
	err := p.SetBarCodeHeight(162)
	if err != nil {
		return fmt.Errorf("could not reset bar code height: %w", err)
	}
	return nil
}

// SetBarCodeHeight sets the bar code height in n dots
func (p Printer) SetBarCodeHeight(n int) error {
	errMsg := "could not set bar code height: %w"

	err := checkRange(n, 1, 255, "height")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{GS, 'h', byte(n)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return nil
}

// SetCharacterSize sets both the width and height of text characters.  The
// values for height and width must be between 0 and 7, inclusively.
//
// While the printer may accept all valid values, it may not print text
// correctly.
func (p Printer) SetCharacterSize(width, height int) error {
	errMsg := "could not set character size: %w"

	err := checkRange(width, 0, 7, "character width")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	err = checkRange(height, 0, 7, "character height")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{GS, '!', uint8(width)<<4 | uint8(height)})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return nil
}

// SetUpsideDown will either set or clear printing text upside-down.  This
// setting does not affect images or barcodes.
func (p Printer) SetUpsideDown(upsidedown bool) error {
	var val uint8 = 0
	if upsidedown {
		val = 1
	}

	fmt.Printf("upside-down %X\n", val)

	_, err := p.Write([]byte{ESC, '{', val})
	if err != nil {
		return fmt.Errorf("could not set upside-down mode: %w", err)
	}
	return nil
}

func checkBarcodeCodabarData(data string) error {
	body := "0123456789-$:/.+"
	wrappers := "ABCD"

	if !strings.ContainsRune(wrappers, rune(data[0])) || !strings.ContainsRune(wrappers, rune(data[len(data)-1])) {
		return fmt.Errorf("the first and last byte of CODABAR must be one of %s", wrappers)
	}

	for _, d := range data {
		if !strings.ContainsRune(body, d) {
			return fmt.Errorf("%s was in the bar code data and only %q is accepted", string(d), body)
		}
	}

	return nil
}

func checkBarcodeData(data, accepted string) error {
	for _, d := range data {
		if !strings.ContainsRune(accepted, d) {
			return fmt.Errorf("%s was in the bar code data and only %q is accepted", string(d), accepted)
		}
	}
	return nil
}

// PrintBarCode prints the bar code passed in with data.
//
// The size ranges are as follows in (Type: min, max):
//
//	BcUPCA: 11, 12
//	BcUPCE: 6, 7
//	BcJAN13: 12, 13
//	BcJAN8: 7, 8
//	BcCODE39: 0, 14
//	BcITF: 0, 22
//	BcCODABAR: 2, 19
//	BcCODE93: 1, 17
//	BcCODE123: 0, 65
//
// For the accepted data values:
//
//	BcUPCA, BcUPCE, BcJAN13, BcJAN8, BcITF all only accept [0123456789]
//	BcCODE39, BcCODE93, BcCODE123 can accept [ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-.*$/+% ]
//	BcCODABAR:
//	  The first and last character of the CODABAR code bar has to be one of [ABCD]
//	  and the rest of the characters in between can be one of [0123456789-$:/.+]
//
// Note on the CODE123 length:
//
//	...the docs say it's between 2 and 255 but the printer
//	does not have that limit. On one hand it can go down to 0 character, but also I could
//	not find the limit for max characters. At 15 characters it went off the page with a
//	HOP-E802 printer and at 34 characters it starts printing the HRI weird. At 66 0s
//	repeating it seems to break and stop printing, and the same at 65 As repeating.
//	Long story short...I think they didn't finish programming the checks on CODE123
func (p Printer) PrintBarCode(barcodeType BarCode, data string) error {
	errMsg := "could not print bar code: %w"

	err := checkEnum(barcodeType, allBarcodes...)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	dataLength := len(data)

	// Check length
	var min, max int
	switch barcodeType {
	case BcUPCA:
		min, max = 11, 12
	case BcUPCE:
		min, max = 6, 7
	case BcJAN13:
		min, max = 12, 13
	case BcJAN8:
		min, max = 7, 8
	case BcCODE39:
		min, max = 0, 14
	case BcITF:
		min, max = 0, 22
	case BcCODABAR:
		min, max = 2, 19
	case BcCODE93:
		min, max = 1, 17
	case BcCODE123:
		// At 66 characters for 'A...' the printer seems to cry
		// for printing all 0s it cried at 65
		// maybe it needs some friends
		min, max = 0, 60
	}

	err = checkRange(dataLength, min, max, "data length")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	// Check data ranges
	switch barcodeType {
	case BcUPCA, BcUPCE, BcJAN13, BcJAN8, BcITF:
		err = checkBarcodeData(data, "0123456789")
	case BcCODE39, BcCODE93, BcCODE123:
		err = checkBarcodeData(data, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-.*$/+% ")
	case BcCODABAR:
		err = checkBarcodeCodabarData(data)
	}

	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	msg := []byte{0x1D, 'k', byte(barcodeType)}

	// Add data
	if inSlice(barcodeType, lengthBarcodes...) {
		// length defined barcode
		msg = append(msg, byte(dataLength))
		msg = append(msg, data...)
	} else {
		// Null ending barcode
		msg = append(msg, data...)
		msg = append(msg, 0)
	}

	_, err = p.Write(msg)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return nil
}

// SelectPrintMode sets the print mode for the printer.
// The modes are as follows and are provided as variadic arguments:
//   - ThinFont: Font selection, 0: Font A (normal), 1: Font B (thin)
//   - Bold: Emphasized mode, 0: Off, 1: On
//     (Emphasized mode just prints the text bold, hence the flag is named 'bold')
//   - DoubleHeight: Double height, 0: Off (normal), 1: On (double height font)
//   - DoubleWidth: Double width, 0: Off (normal), 1: On (double width font)
//   - Underline: Underline, 0: Off (normal), 1: On (underlined font)
func (p Printer) SelectPrintMode(modes ...PrintModeMask) error {
	errMsg := "could not select print mode: %w"
	var mask byte
	for _, mode := range modes {
		mask |= byte(mode)
	}
	_, err := p.Write([]byte{ESC, '!', mask})
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}
