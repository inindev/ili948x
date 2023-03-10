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
	trans  iTransport
	cs     machine.Pin // spi chip select
	dc     machine.Pin // tft data / command
	bl     machine.Pin // tft backlight
	rst    machine.Pin // tft reset
	width  uint16      // tft pixel width
	height uint16      // tft pixel height
	rot    Rotation    // tft orientation
	mirror bool        // mirror tft output
	bgr    bool        // tft blue-green-red mode
	x0, x1 uint16      // current address window for
	y0, y1 uint16      //  CMD_PASET and CMD_CASET
}

func NewIli9488(trans iTransport, cs, dc, bl, rst machine.Pin, width, height uint16) *Ili948x {
	if width == 0 {
		width = TFT_DEFAULT_WIDTH
	}
	if height == 0 {
		height = TFT_DEFAULT_HEIGHT
	}

	disp := &Ili948x{
		trans:  trans,
		cs:     cs,
		dc:     dc,
		bl:     bl,
		rst:    rst,
		width:  width,
		height: height,
		rot:    Rot_0,
		mirror: false,
		bgr:    false,
		x0:     0,
		x1:     0,
		y0:     0,
		y1:     0,
	}

	// chip select pin
	if cs != machine.NoPin { // cs may be implemented by hardware spi
		cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
		cs.High()
	}

	// data/command pin
	dc.Configure(machine.PinConfig{Mode: machine.PinOutput})
	dc.High()

	// backlight pin
	if bl != machine.NoPin {
		bl.Configure(machine.PinConfig{Mode: machine.PinOutput})
		bl.Low() // display off
	}

	// reset pin
	if rst != machine.NoPin {
		disp.rst.Configure(machine.PinConfig{Mode: machine.PinOutput})
		disp.rst.High()
	}

	// reset the display
	disp.Reset()

	// init display settings
	disp.init()

	// display backlight on
	disp.SetBacklight(true)

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

	disp.writeCmd(CMD_RAMWR)
	disp.startWrite()
	disp.trans.write24n(color, int(width)*int(height))
	disp.endWrite()

	return nil
}

// DisplayBitmap renders the streamed image at given coordinates and dimensions.
func (disp *Ili948x) DisplayBitmap(x, y, width, height uint16, bpp uint8, r io.Reader) error {
	w, h := disp.Size()
	if x >= w || (x+width) > w || y >= h || (y+height) > h {
		return errors.New("rectangle coordinates outside display area")
	}
	disp.setWindow(x, y, width, height)

	disp.writeCmd(CMD_RAMWR)
	buf := make([]uint8, width*uint16(bpp/3))
	for {
		n, err := r.Read(buf)
		if n == 0 || err == io.EOF {
			break
		}

		disp.startWrite()
		disp.trans.write8sl(buf[:n])
		disp.endWrite()
	}

	return nil
}

// SetScrollArea sets an area to scroll with fixed top/bottom or left/right parts of the display
// Rotation affects scroll direction
func (disp *Ili948x) SetScrollArea(topFixedArea, bottomFixedArea uint16) {
	vertScrollArea := disp.height - topFixedArea - bottomFixedArea
	disp.writeCmd(CMD_VSCRDEF,
		uint8(topFixedArea>>8),
		uint8(topFixedArea),
		uint8(vertScrollArea>>8),
		uint8(vertScrollArea),
		uint8(bottomFixedArea>>8),
		uint8(bottomFixedArea))
}

// SetScroll sets the vertical scroll address of the display.
func (disp *Ili948x) SetScroll(line uint16) {
	disp.writeCmd(CMD_VSCRSADD,
		uint8(line>>8),
		uint8(line))
}

// StopScroll returns the display to its normal state
func (disp *Ili948x) StopScroll() {
	disp.writeCmd(CMD_NORON)
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

// GetMirror returns true if the display set to display a mirrored image.
func (disp *Ili948x) GetMirror() bool {
	return disp.mirror
}

// SetMirror switches the display between mirrored image and non-mirrored image mode.
func (disp *Ili948x) SetMirror(mirror bool) {
	disp.mirror = mirror
	disp.updateMadctl()
}

// GetBGR returns true if the display is in blue-green-red (BGR) mode.
func (disp *Ili948x) GetBGR() bool {
	return disp.bgr
}

// SetBGR switches the display between blue-green-red (BGR) and red-green-blue (RGB) mode.
func (disp *Ili948x) SetBGR(bgr bool) {
	disp.bgr = bgr
	disp.updateMadctl()
}

// SetBacklight turns the TFT backlight on / off.
func (disp *Ili948x) SetBacklight(b bool) {
	if disp.bl != machine.NoPin {
		disp.bl.Set(b)
	}
}

// Reset performs a hardware reset if rst pin present, otherwise performs a CMD_SWRESET software reset of the TFT display.
func (disp *Ili948x) Reset() {
	// prefer a hardware reset if there is one
	if disp.rst != machine.NoPin {
		disp.rst.Low()
		time.Sleep(time.Millisecond * 64) // datasheet says 10ms
		disp.rst.High()
	} else {
		// if no hardware reset, send software reset
		disp.writeCmd(CMD_SWRESET)
	}
	time.Sleep(time.Millisecond * 140) // datasheet says 120ms
}

// setWindow defines the output area for subsequent calls to CMD_RAMWR
func (disp *Ili948x) setWindow(x, y, w, h uint16) {
	x1 := x + w - 1
	if x != disp.x0 || x1 != disp.x1 {
		disp.writeCmd(CMD_CASET,
			uint8(x>>8),
			uint8(x),
			uint8(x1>>8),
			uint8(x1),
		)
		disp.x0, disp.x1 = x, x1
	}
	y1 := y + h - 1
	if y != disp.y0 || y1 != disp.y1 {
		disp.writeCmd(CMD_PASET,
			uint8(y>>8),
			uint8(y),
			uint8(y1>>8),
			uint8(y1),
		)
		disp.y0, disp.y1 = y, y1
	}
}

// updateMadctl updates CMD_MADCTRL based settings (mirror, rotation, RGB/BGR)
func (disp *Ili948x) updateMadctl() {
	madctl := uint8(0)

	if !disp.mirror {
		// regular
		switch disp.rot {
		case Rot_0:
			madctl = 0
		case Rot_90:
			madctl = MADCTRL_MX | MADCTRL_MH | MADCTRL_MV
		case Rot_180:
			madctl = MADCTRL_MX | MADCTRL_MH | MADCTRL_MY | MADCTRL_ML
		case Rot_270:
			madctl = MADCTRL_MV | MADCTRL_MY | MADCTRL_ML
		}
	} else {
		// mirrored
		switch disp.rot {
		case Rot_0:
			madctl = MADCTRL_MX | MADCTRL_MH
		case Rot_90:
			madctl = MADCTRL_MX | MADCTRL_MH | MADCTRL_MY | MADCTRL_ML | MADCTRL_MV
		case Rot_180:
			madctl = MADCTRL_MY | MADCTRL_ML
		case Rot_270:
			madctl = MADCTRL_MV
		}
	}

	if disp.bgr {
		madctl |= MADCTRL_BGR
	}

	disp.writeCmd(CMD_MADCTRL, madctl)
}

// init performs base-level initialization and setup of the TFT display
func (disp *Ili948x) init() {
	disp.writeCmd(CMD_PWCTRL1,
		0x17, // VREG1OUT:  5.0000
		0x15) // VREG2OUT: -4.8750

	disp.writeCmd(CMD_PWCTRL2,
		0x41) // VGH: VCI x 6  VGL: -VCI x 4

	disp.writeCmd(CMD_VMCTRL,
		0x00, // nVM
		0x12, // VCM_REG:    -1.71875
		0x80, // VCM_REG_EN: true
		0x40) // VCM_OUT

	disp.writeCmd(CMD_PIXFMT,
		0x66) // DPI/DBI: 18 bits / pixel

	disp.writeCmd(CMD_FRMCTRL1,
		0xa0, // FRS: 60.76  DIVA: 0
		0x11) // RTNA: 17 clocks

	disp.writeCmd(CMD_INVCTRL,
		0x02) // DINV: 2 dot inversion

	disp.writeCmd(CMD_DISCTRL,
		0x02, // PT: AGND
		0x22, // SS: S960 -> S1  ISC: 5 frames
		0x3b) // NL: 8 * (3b + 1) = 480 lines

	disp.writeCmd(CMD_ETMOD,
		0xc6) // EPF: 11 (db5 -> r0,g0,b0)

	disp.writeCmd(CMD_ADJCTRL3,
		0xa9, //
		0x51, //
		0x2c, //
		0x82) // DSI_18_option:

	disp.updateMadctl()

	disp.writeCmd(CMD_SLPOUT)
	time.Sleep(time.Millisecond * 120)
	disp.writeCmd(CMD_DISON)
}

// writeCmd issues a TFT command with optional data
func (disp *Ili948x) writeCmd(cmd uint8, data ...uint8) {
	disp.startWrite()

	disp.dc.Low() // command mode
	disp.trans.write8(cmd)

	disp.dc.High() // data mode
	disp.trans.write8sl(data)

	disp.endWrite()
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
