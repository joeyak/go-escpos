package escpos

import "fmt"

func (p Printer) realTimeStatusTransmission(n int) (byte, error) {
	errMsg := "could not transmit real-time status: %w"

	err := checkRange(n, 1, 4, "n status type")
	if err != nil {
		return 0, fmt.Errorf(errMsg, err)
	}

	_, err = p.Write([]byte{DLE, 0x04, byte(n)})
	if err != nil {
		return 0, fmt.Errorf(errMsg, err)
	}

	b := make([]byte, 1)
	_, err = p.Read(b)
	if err != nil {
		return 0, fmt.Errorf(errMsg, err)
	}

	return b[0], nil
}

type PrinterStatus struct {
	DrawerOpen bool
}

func (p Printer) TransmitPrinterStatus() (PrinterStatus, error) {
	b, err := p.realTimeStatusTransmission(1)
	if err != nil {
		return PrinterStatus{}, fmt.Errorf("could not transmit printer status: %w", err)
	}

	return PrinterStatus{
		DrawerOpen: b&0b0100 == 0b0100,
	}, nil
}

type OfflineStatus struct {
	CoverOpen, FeedButton, PrintingStopped, ErrorOccured bool
}

func (p Printer) TransmitOfflineStatus() (OfflineStatus, error) {
	b, err := p.realTimeStatusTransmission(2)
	if err != nil {
		return OfflineStatus{}, fmt.Errorf("could not transmit offline status: %w", err)
	}

	return OfflineStatus{
		CoverOpen:       b&0b0000_0100 == 0b0000_0100,
		FeedButton:      b&0b0000_1000 == 0b0000_1000,
		PrintingStopped: b&0b0010_0000 == 0b0010_0000,
		ErrorOccured:    b&0b0100_0000 == 0b0100_0000,
	}, nil
}

type ErrorStatus struct {
	AutoCutter, UnRecoverable, AutoRecoverable bool
}

func (p Printer) TransmitErrorStatus() (ErrorStatus, error) {
	b, err := p.realTimeStatusTransmission(3)
	if err != nil {
		return ErrorStatus{}, fmt.Errorf("could not transmit error status: %w", err)
	}

	return ErrorStatus{
		AutoCutter:      b&0b0000_1000 == 0b0000_1000,
		UnRecoverable:   b&0b0010_0000 == 0b0010_0000,
		AutoRecoverable: b&0b0100_0000 == 0b0100_0000,
	}, nil
}

type PaperSensorStatus struct {
	NearEnd, RollEnd bool
}

func (p Printer) TransmitPaperSensorStatus() (PaperSensorStatus, error) {
	b, err := p.realTimeStatusTransmission(4)
	if err != nil {
		return PaperSensorStatus{}, fmt.Errorf("could not transmit error status: %w", err)
	}

	return PaperSensorStatus{
		NearEnd: b&0b0000_1100 == 0b0000_1100,
		RollEnd: b&0b0110_0000 == 0b0110_0000,
	}, nil
}
