package main

import (
	"errors"
	"io"
	"machine"
	"time"
)

type Rotation uint8

const ( // clock-wise rotation
	Rot_0   Rotation = iota // i.e. 320x480
	Rot_90                  // i.e. 480x320
	Rot_180                 // i.e. 320x480
	Rot_270                 // i.e. 480x320
)

type TFTPanel struct {
	tspt   iTransport
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

func NewTFTPanel(tspt iTransport, cs, dc, bl, rst machine.Pin, width, height uint16) *TFTPanel {
	if width == 0 {
		width = TFT_DEFAULT_WIDTH
	}
	if height == 0 {
		height = TFT_DEFAULT_HEIGHT
	}

	tft := &TFTPanel{
		tspt:   tspt,
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
		rst.Configure(machine.PinConfig{Mode: machine.PinOutput})
		rst.High()
	}

	// reset the display
	tft.Reset()

	// init display settings
	tft.initPanel()

	// display backlight on
	tft.SetBacklight(true)

	return tft
}

// Size returns the current size of the display.
func (tft *TFTPanel) Size() (uint16, uint16) {
	if tft.rot == Rot_0 || tft.rot == Rot_180 {
		return tft.width, tft.height
	}
	return tft.height, tft.width
}

// DrawPixel draws a single pixel with the specified color.
func (tft *TFTPanel) DrawPixel(x, y uint16, color uint32) error {
	return tft.FillRectangle(x, y, 1, 1, color)
}

// DrawHLine draws a horizontal line with the specified color.
func (tft *TFTPanel) DrawHLine(x0, x1, y uint16, color uint32) error {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	return tft.FillRectangle(x0, y, x1-x0+1, 1, color)
}

// DrawVLine draws a vertical line with the specified color.
func (tft *TFTPanel) DrawVLine(x, y0, y1 uint16, color uint32) error {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return tft.FillRectangle(x, y0, 1, y1-y0+1, color)
}

// FillScreen fills the screen with the specified color.
func (tft *TFTPanel) FillScreen(color uint32) {
	if tft.rot == Rot_0 || tft.rot == Rot_180 {
		tft.FillRectangle(0, 0, tft.width, tft.height, color)
	} else {
		tft.FillRectangle(0, 0, tft.height, tft.width, color)
	}
}

// FillRectangle fills a rectangle at given coordinates and dimensions with the specified color.
func (tft *TFTPanel) FillRectangle(x, y, width, height uint16, color uint32) error {
	w, h := tft.Size()
	if x >= w || (x+width) > w || y >= h || (y+height) > h {
		return errors.New("rectangle coordinates outside display area")
	}
	tft.setWindow(x, y, width, height)

	tft.writeCmd(CMD_RAMWR)
	tft.startWrite()
	tft.tspt.write24n(color, int(width)*int(height))
	tft.endWrite()

	return nil
}

// DisplayBitmap renders the streamed image at given coordinates and dimensions.
func (tft *TFTPanel) DisplayBitmap(x, y, width, height uint16, bpp uint8, r io.Reader) error {
	w, h := tft.Size()
	if x >= w || (x+width) > w || y >= h || (y+height) > h {
		return errors.New("rectangle coordinates outside display area")
	}
	tft.setWindow(x, y, width, height)

	tft.writeCmd(CMD_RAMWR)
	buf := make([]uint8, width*uint16(bpp/3))
	for {
		n, err := r.Read(buf)
		if n == 0 || err == io.EOF {
			break
		}

		tft.startWrite()
		tft.tspt.write8sl(buf[:n])
		tft.endWrite()
	}

	return nil
}

// SetScrollArea sets an area to scroll with fixed top/bottom or left/right parts of the display
// Rotation affects scroll direction
func (tft *TFTPanel) SetScrollArea(topFixedArea, bottomFixedArea uint16) {
	vertScrollArea := tft.height - topFixedArea - bottomFixedArea
	tft.writeCmd(CMD_VSCRDEF,
		uint8(topFixedArea>>8),
		uint8(topFixedArea),
		uint8(vertScrollArea>>8),
		uint8(vertScrollArea),
		uint8(bottomFixedArea>>8),
		uint8(bottomFixedArea))
}

// SetScroll sets the vertical scroll address of the display.
func (tft *TFTPanel) SetScroll(line uint16) {
	tft.writeCmd(CMD_VSCRSADD,
		uint8(line>>8),
		uint8(line))
}

// StopScroll returns the display to its normal state
func (tft *TFTPanel) StopScroll() {
	tft.writeCmd(CMD_NORON)
}

// GetRotation returns the current rotation of the display.
func (tft *TFTPanel) GetRotation() Rotation {
	return tft.rot
}

// SetRotation sets the clock-wise rotation of the display.
func (tft *TFTPanel) SetRotation(rot Rotation) {
	tft.rot = rot
	tft.updateMadctl()
}

// GetMirror returns true if the display set to display a mirrored image.
func (tft *TFTPanel) GetMirror() bool {
	return tft.mirror
}

// SetMirror switches the display between mirrored image and non-mirrored image mode.
func (tft *TFTPanel) SetMirror(mirror bool) {
	tft.mirror = mirror
	tft.updateMadctl()
}

// GetBGR returns true if the display is in blue-green-red (BGR) mode.
func (tft *TFTPanel) GetBGR() bool {
	return tft.bgr
}

// SetBGR switches the display between blue-green-red (BGR) and red-green-blue (RGB) mode.
func (tft *TFTPanel) SetBGR(bgr bool) {
	tft.bgr = bgr
	tft.updateMadctl()
}

// SetBacklight turns the TFT backlight on / off.
func (tft *TFTPanel) SetBacklight(b bool) {
	if tft.bl != machine.NoPin {
		tft.bl.Set(b)
	}
}

// Reset performs a hardware reset if rst pin present, otherwise performs a CMD_SWRESET software reset of the TFT display.
func (tft *TFTPanel) Reset() {
	// prefer a hardware reset if there is one
	if tft.rst != machine.NoPin {
		tft.rst.Low()
		time.Sleep(time.Millisecond * 64) // datasheet says 10ms
		tft.rst.High()
	} else {
		// if no hardware reset, send software reset
		tft.writeCmd(CMD_SWRESET)
	}
	time.Sleep(time.Millisecond * 140) // datasheet says 120ms
}

// setWindow defines the output area for subsequent calls to CMD_RAMWR
func (tft *TFTPanel) setWindow(x, y, w, h uint16) {
	x1 := x + w - 1
	if x != tft.x0 || x1 != tft.x1 {
		tft.writeCmd(CMD_CASET,
			uint8(x>>8),
			uint8(x),
			uint8(x1>>8),
			uint8(x1),
		)
		tft.x0, tft.x1 = x, x1
	}
	y1 := y + h - 1
	if y != tft.y0 || y1 != tft.y1 {
		tft.writeCmd(CMD_PASET,
			uint8(y>>8),
			uint8(y),
			uint8(y1>>8),
			uint8(y1),
		)
		tft.y0, tft.y1 = y, y1
	}
}

// updateMadctl updates CMD_MADCTRL based settings (mirror, rotation, RGB/BGR)
func (tft *TFTPanel) updateMadctl() {
	madctl := uint8(0)

	if !tft.mirror {
		// regular
		switch tft.rot {
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
		switch tft.rot {
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

	if tft.bgr {
		madctl |= MADCTRL_BGR
	}

	tft.writeCmd(CMD_MADCTRL, madctl)
}

// writeCmd issues a TFT command with optional data
func (tft *TFTPanel) writeCmd(cmd uint8, data ...uint8) {
	tft.startWrite()

	tft.dc.Low() // command mode
	tft.tspt.write8(cmd)

	tft.dc.High() // data mode
	tft.tspt.write8sl(data)

	tft.endWrite()
}

//go:inline
func (tft *TFTPanel) startWrite() {
	if tft.cs != machine.NoPin {
		tft.cs.Low()
	}
}

//go:inline
func (tft *TFTPanel) endWrite() {
	if tft.cs != machine.NoPin {
		tft.cs.High()
	}
}
