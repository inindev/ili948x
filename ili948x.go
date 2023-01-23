package main

import (
	"errors"
	"io"
	"machine"
	"time"
)

type Rotation uint8

const ( // clock-wise rotation
	Rot_0   Rotation = iota // 320x480
	Rot_90                  // 480x320
	Rot_180                 // 320x480
	Rot_270                 // 480x320
)

const (
	TFT_DEFAULT_WIDTH  uint16 = 320 // rot_0
	TFT_DEFAULT_HEIGHT uint16 = 480
)

type Ili948x struct {
	spi      machine.SPI // spi bus
	cs       machine.Pin // spi chip select
	dc       machine.Pin // tft data / command
	bl       machine.Pin // tft backlight
	spiTxBuf []byte      // spi tx buffer
	width    uint16      // tft pixel width
	height   uint16      // tft pixel height
	rot      Rotation    // tft orientation
	mirror   bool        // mirror tft output
	bgr      bool        // tft blue-green-red mode
	x0, x1   uint16      // current address window for
	y0, y1   uint16      //  CMD_PASET and CMD_CASET
}

func NewIli9488(spi machine.SPI, cs, dc, bl, rst machine.Pin, spiTxBuf []byte, width, height uint16) *Ili948x {

	if width == 0 {
		width = TFT_DEFAULT_WIDTH
	}
	if height == 0 {
		height = TFT_DEFAULT_HEIGHT
	}

	disp := &Ili948x{
		spi:      spi,
		cs:       cs,
		dc:       dc,
		bl:       bl,
		spiTxBuf: spiTxBuf,
		width:    width,
		height:   height,
		rot:      Rot_0,
		mirror:   false,
		bgr:      false,
		x0:       0,
		x1:       0,
		y0:       0,
		y1:       0,
	}

	// chip select pin
	if cs != machine.NoPin { // cs may be implemented by hardware spi
		cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
		cs.High()
	}

	// backlight pin
	if bl != machine.NoPin {
		bl.Configure(machine.PinConfig{Mode: machine.PinOutput})
		bl.Low() // display off
	}

	// data/command pin
	dc.Configure(machine.PinConfig{Mode: machine.PinOutput})
	dc.High()

	// reset the display
	if rst != machine.NoPin {
		// configure hardware reset if there is one
		rst.Configure(machine.PinConfig{Mode: machine.PinOutput})
		rst.High()
		time.Sleep(time.Millisecond * 100)
		rst.Low()
		time.Sleep(time.Millisecond * 100)
		rst.High()
		time.Sleep(time.Millisecond * 200)
	} else {
		// if no hardware reset, send software reset
		disp.writeCmd(CMD_SWRESET, nil)
		time.Sleep(time.Millisecond * 150)
	}

	disp.init()

	if bl != machine.NoPin {
		bl.High() // display on
	}

	return disp
}

// Size returns the current size of the display.
func (disp *Ili948x) Size() (uint16, uint16) {
	if disp.rot == Rot_0 || disp.rot == Rot_180 {
		return disp.width, disp.height
	}
	return disp.height, disp.width
}

// DrawPixel draws a single pixel with the specified color.
func (disp *Ili948x) DrawPixel(x, y uint16, color uint32) error {
	return disp.FillRectangle(x, y, 1, 1, color)
}

// DrawHLine draws a horizontal line with the specified color.
func (disp *Ili948x) DrawHLine(x0, x1, y uint16, color uint32) error {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	return disp.FillRectangle(x0, y, x1-x0+1, 1, color)
}

// DrawVLine draws a vertical line with the specified color.
func (disp *Ili948x) DrawVLine(x, y0, y1 uint16, color uint32) error {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return disp.FillRectangle(x, y0, 1, y1-y0+1, color)
}

// FillScreen fills the screen with the specified color.
func (disp *Ili948x) FillScreen(color uint32) {
	if disp.rot == Rot_0 || disp.rot == Rot_180 {
		disp.FillRectangle(0, 0, disp.width, disp.height, color)
	} else {
		disp.FillRectangle(0, 0, disp.height, disp.width, color)
	}
}

// FillRectangle fills a rectangle at given coordinates and dimensions with the specified color.
func (disp *Ili948x) FillRectangle(x, y, width, height uint16, color uint32) error {
	w, h := disp.Size()
	if x >= w || (x+width) > w || y >= h || (y+height) > h {
		return errors.New("rectangle coordinates outside display area")
	}
	disp.setWindow(x, y, width, height)

	disp.writeCmdRepeat(
		CMD_RAMWR,
		[]byte{
			uint8(color),        // B
			uint8(color >> 8),   // G
			uint8(color >> 16)}, // R
		uint32(width)*uint32(height))

	return nil
}

// DisplayBitmap renders the streamed image at given coordinates and dimensions.
func (disp *Ili948x) DisplayBitmap(x, y, width, height uint16, bpp uint8, r io.Reader) error {
	w, h := disp.Size()
	if x >= w || (x+width) > w || y >= h || (y+height) > h {
		return errors.New("rectangle coordinates outside display area")
	}
	disp.setWindow(x, y, width, height)

	disp.writeCmd(CMD_RAMWR, nil)
	buf := make([]byte, width*uint16(bpp/3))
	for {
		n, err := r.Read(buf)
		disp.writeData(buf)
		if n == 0 || err == io.EOF {
			break
		}
	}

	return nil
}

// GetBGR returns true is the display is in blue-green-red (BGR) mode.
func (disp *Ili948x) GetBGR() bool {
	return disp.bgr
}

// SetBGR switches the display between blue-green-red (BGR) and red-green-blue (RGB) mode.
func (disp *Ili948x) SetBGR(bgr bool) {
	disp.bgr = bgr
	disp.updateMadctl()
}

// GetRotation returns the current rotation of the display.
func (disp *Ili948x) GetRotation() Rotation {
	return disp.rot
}

// SetRotation sets the clock-wise rotation of the display.
func (disp *Ili948x) SetRotation(rot Rotation) {
	disp.rot = rot
	disp.updateMadctl()
}

func (disp *Ili948x) updateMadctl() {
	madctl := uint8(0)

	switch disp.rot {
	case Rot_0:
		madctl = 0
	case Rot_90:
		madctl = MADCTRL_MV | MADCTRL_MX | MADCTRL_MH
	case Rot_180:
		madctl = MADCTRL_MX | MADCTRL_MH | MADCTRL_MY | MADCTRL_ML
	case Rot_270:
		madctl = MADCTRL_MV | MADCTRL_MY | MADCTRL_ML

		// case Rotation0Mirror:
		// 	madctl = MADCTRL_BGR
		// case Rotation90Mirror:
		// 	madctl = MADCTRL_MY | MADCTRL_MV | MADCTRL_BGR
		// case Rotation180Mirror:
		// 	madctl = MADCTRL_MX | MADCTRL_MY | MADCTRL_BGR
		// case Rotation270Mirror:
		// 	madctl = MADCTRL_MX | MADCTRL_MY | MADCTRL_MV | MADCTRL_BGR
	}

	if disp.bgr {
		madctl |= MADCTRL_BGR
	}

	disp.writeCmd(CMD_MADCTRL, []byte{madctl})
}

// setWindow defines the output area for subsequent calls to CMD_RAMWR
func (disp *Ili948x) setWindow(x, y, w, h uint16) {
	x1 := x + w - 1
	if x != disp.x0 || x1 != disp.x1 {
		disp.writeCmd(CMD_CASET, []byte{
			byte(x >> 8),
			byte(x),
			byte(x1 >> 8),
			byte(x1),
		})
		disp.x0, disp.x1 = x, x1
	}
	y1 := y + h - 1
	if y != disp.y0 || y1 != disp.y1 {
		disp.writeCmd(CMD_PASET, []byte{
			byte(y >> 8),
			byte(y),
			byte(y1 >> 8),
			byte(y1),
		})
		disp.y0, disp.y1 = y, y1
	}
}

//go:inline
func (disp *Ili948x) startWrite() {
	if disp.cs != machine.NoPin {
		disp.cs.Low()
	}
}

//go:inline
func (disp *Ili948x) endWrite() {
	if disp.cs != machine.NoPin {
		disp.cs.High()
	}
}

// ////////////////////////////////////
func (disp *Ili948x) writeCmd(cmd byte, data []byte) {
	disp.startWrite()

	disp.dc.Low() // command mode
	disp.spi.Transfer(cmd)
	disp.dc.High() // data mode

	if data != nil {
		disp.spi.Tx(data, nil)
	}

	disp.endWrite()
}

// ////////////////////////////////////
func (disp *Ili948x) writeCmdRepeat(cmd byte, data []byte, reps uint32) {
	disp.startWrite()

	disp.dc.Low() // command mode
	disp.spi.Transfer(cmd)

	disp.dc.High() // data mode
	dataBytes := uint32(len(data))
	txBufBytes := uint32(len(disp.spiTxBuf))
	if reps < 1 || dataBytes < 1 || txBufBytes < 1 {
		for i := uint32(0); i < reps; i++ {
			disp.spi.Tx(data, nil)
		}
	} else {
		chunks := txBufBytes / dataBytes // 64 / 3 = 21
		bufBytes := chunks * dataBytes
		for i := uint32(0); i < bufBytes; i++ {
			disp.spiTxBuf[i] = data[i%dataBytes]
		}

		q := reps / chunks
		for i := uint32(0); i < q; i++ {
			disp.spi.Tx(disp.spiTxBuf[:bufBytes], nil)
		}
		r := reps % chunks
		disp.spi.Tx(disp.spiTxBuf[:r*dataBytes], nil)
	}

	disp.endWrite()
}

// func (disp *Ili948x) writeCmdRepeat(cmd byte, data []byte, reps uint32) {
// 	disp.startWrite()
//
// 	disp.dc.Low() // command mode
// 	disp.spi.Transfer(cmd)
//
// 	disp.dc.High() // data mode
// 	for i := uint32(0); i < reps; i++ {
// 		disp.spi.Tx(data, nil)
// 	}
//
// 	disp.endWrite()
// }

// ////////////////////////////////////
func (disp *Ili948x) writeData(data []byte) {
	disp.startWrite()
	machine.SPI2.Tx(data, nil)
	disp.endWrite()
}

// ////////////////////////////////////
func (disp *Ili948x) init() {
	disp.writeCmd(CMD_PWCTRL1, []byte{
		0x17,  // VREG1OUT:  5.0000
		0x15}) // VREG2OUT: -4.8750

	disp.writeCmd(CMD_PWCTRL2, []byte{
		0x41}) // VGH: VCI x 6  VGL: -VCI x 4

	disp.writeCmd(CMD_VMCTRL, []byte{
		0x00,  // nVM
		0x12,  // VCM_REG:    -1.71875
		0x80,  // VCM_REG_EN: true
		0x40}) // VCM_OUT

	// TODO: is this correct?
	disp.writeCmd(CMD_PIXFMT, []byte{
		//		0x66}) // DPI/DBI: 18 bits / pixel
		0x76}) // DPI/DBI: 24 bits / pixel

	disp.writeCmd(CMD_FRMCTRL1, []byte{
		0xa0,  // FRS: 60.76  DIVA: 0
		0x11}) // RTNA: 17 clocks

	disp.writeCmd(CMD_INVCTRL, []byte{
		0x02}) // DINV: 2 dot inversion

	disp.writeCmd(CMD_DISCTRL, []byte{
		0x02,  // PT: AGND
		0x22,  // SS: S960 -> S1  ISC: 5 frames
		0x3b}) // NL: 8 * (3b + 1) = 480 lines

	disp.writeCmd(CMD_ETMOD, []byte{
		0xc6}) // EPF: 11 (db5 -> r0,g0,b0)

	disp.writeCmd(CMD_ADJCTRL3, []byte{
		0xa9,  //
		0x51,  //
		0x2c,  //
		0x82}) // DSI_18_option:

	disp.updateMadctl()

	disp.writeCmd(CMD_SLPOUT, nil)
	time.Sleep(time.Millisecond * 120)
	disp.writeCmd(CMD_IDMOFF, nil)
	disp.writeCmd(CMD_DISON, nil)
	time.Sleep(time.Millisecond * 100)
}