//go:build tft_ili9341

package main

import (
	"time"
)

const (
	TFT_DEFAULT_WIDTH  uint16 = 240 // rot_0
	TFT_DEFAULT_HEIGHT uint16 = 320
)

// init performs base-level initialization and setup of the Ili948x TFT display
func (tft *TFTPanel) initPanel() {
	tft.writeCmd(CMD_PWCTRL1, 0x23)             // 4.80V
	tft.writeCmd(CMD_PWCTRL2, 0x10)             // DDVDH: VCIx2
	tft.writeCmd(CMD_PWCTRLB, 0x00, 0xC1, 0x30) // DRV_ena

	tft.writeCmd(CMD_PWSEQCTRL, 0x64, 0x03, 0x12, 0x81)
	tft.writeCmd(CMD_VMCTRL1, 0x3e, 0x28) // VMH: 5.850V, VML: -1.500V
	tft.writeCmd(CMD_VMCTRL2, 0x86)       // VMF: VMH–58, VML–58

	tft.writeCmd(CMD_TIMCTRLA_INT, 0x85, 0x00, 0x78)
	tft.writeCmd(CMD_PUMPRATIO, 0x20)      // DDVDH=2xVCI
	tft.writeCmd(CMD_TIMCTRLB, 0x00, 0x00) // VG_SW_T1:EQ to GND, VG_SW_T2:EQ to DDVDH, VG_SW_T3:EQ to DDVDH, VG_SW_T4:EQ to GND

	tft.writeCmd(CMD_PIXFMT,
		0x66) // DPI/DBI: 18 bits / pixel

	tft.writeCmd(CMD_INVON)

	tft.writeCmd(CMD_GAM3CTRL, 0x00)
	tft.writeCmd(CMD_GAMSET, 0x01) // Gamma set, curve 1
	tft.writeCmd(CMD_GAMCTRLP, 0x0F, 0x31, 0x2B, 0x0C, 0x0E, 0x08, 0x4E, 0xF1, 0x37, 0x07, 0x10, 0x03, 0x0E, 0x09, 0x00)
	tft.writeCmd(CMD_GAMCTRLN, 0x00, 0x0E, 0x14, 0x03, 0x11, 0x07, 0x31, 0xC1, 0x48, 0x08, 0x0F, 0x0C, 0x31, 0x36, 0x0F)
	tft.writeCmd(CMD_DISCTRL, 0x08, 0xC2, 0x27)

	tft.updateMadctl()

	tft.writeCmd(CMD_SLPOUT)
	time.Sleep(time.Millisecond * 120)
	tft.writeCmd(CMD_DISON)
}

const ( // ILI9341 Datasheet, pp. 83-88
	CMD_NOP     uint8 = 0x00 // No Operation
	CMD_SWRESET       = 0x01 // Software Reset

	CMD_RDDIDIF    = 0x04 // Read Display Identification Information
	CMD_RDDST      = 0x09 // Read Display Status
	CMD_RDDPM      = 0x0a // Read Display Power Mode
	CMD_RDDMADCTRL = 0x0b // Read Display MADCTRL
	CMD_RDDCOLMOD  = 0x0c // Read Display Pixel Format
	CMD_RDDIM      = 0x0d // Read Display Image Mode
	CMD_RDDSM      = 0x0e // Read Display Signal Mode
	CMD_RDDSDR     = 0x0f // Read Display Self-Diagnostic Result

	CMD_SLPIN  = 0x10 // Enter Sleep Mode
	CMD_SLPOUT = 0x11 // Sleep Out
	CMD_PTLON  = 0x12 // Partial Mode ON
	CMD_NORON  = 0x13 // Normal Display Mode ON
	CMD_INVOFF = 0x20 // Display Inversion OFF
	CMD_INVON  = 0x21 // Display Inversion ON
	CMD_GAMSET = 0x26 // Gamma Set
	CMD_DISOFF = 0x28 // Display OFF
	CMD_DISON  = 0x29 // Display ON

	CMD_CASET    = 0x2a // Column Address Set
	CMD_PASET    = 0x2b // Page Address Set
	CMD_RAMWR    = 0x2c // Memory Write
	CMD_RGBSET   = 0x2d // Color Set
	CMD_RAMRD    = 0x2e // Memory Read
	CMD_PLTAR    = 0x30 // Partial Area
	CMD_VSCRDEF  = 0x33 // Vertical Scrolling Definition
	CMD_TEOFF    = 0x34 // Tearing Effect Line OFF
	CMD_TEON     = 0x35 // Tearing Effect Line ON
	CMD_MADCTRL  = 0x36 // Memory Access Control
	CMD_VSCRSADD = 0x37 // Vertical Scrolling Start Address
	CMD_IDMOFF   = 0x38 // Idle Mode OFF
	CMD_IDMON    = 0x39 // Idle Mode ON
	CMD_PIXFMT   = 0x3a // COLMOD: Interface Pixel Format
	CMD_RAMWRC   = 0x3c // Memory Write Continue
	CMD_RAMRDRC  = 0x3e // Memory Read Continue
	CMD_TESLWR   = 0x44 // Write Tear Scan Line
	CMD_TESLRD   = 0x45 // Read Tear Scan Line
	CMD_WRDISBV  = 0x51 // Write Display Brightness Value
	CMD_RDDISBV  = 0x52 // Read Display Brightness Value
	CMD_WRCTRLD  = 0x53 // Write CTRL Display Value
	CMD_RDCTRLD  = 0x54 // Read CTRL Display Value
	CMD_WRCABC   = 0x55 // Write Content Adaptive Brightness Control Value
	CMD_RDCABC   = 0x56 // Read Content Adaptive Brightness Control Value
	CMD_WRCABCMB = 0x5e // Write CABC Minimum Brightness
	CMD_RDCABCMB = 0x5f // Read CABC Minimum Brightness

	CMD_IFMODE   = 0xb0 // RGB Interface Signal Control
	CMD_FRMCTRL1 = 0xb1 // Frame Rate Control (In Normal Mode/Full Colors)
	CMD_FRMCTRL2 = 0xb2 // Frame Rate Control (In Idle Mode/8 colors)
	CMD_FRMCTRL3 = 0xb3 // Frame Rate control (In Partial Mode/Full Colors)
	CMD_INVCTRL  = 0xb4 // Display Inversion Control
	CMD_PRCTRL   = 0xb5 // Blanking Porch Control
	CMD_DISCTRL  = 0xb6 // Display Function Control
	CMD_ETMOD    = 0xb7 // Entry Mode Set
	CMD_BKLCTRL1 = 0xb8 // Backlight Control 1
	CMD_BKLCTRL2 = 0xb9 // Backlight Control 2
	CMD_BKLCTRL3 = 0xba // Backlight Control 3
	CMD_BKLCTRL4 = 0xbb // Backlight Control 4
	CMD_BKLCTRL5 = 0xbc // Backlight Control 5
	CMD_BKLCTRL7 = 0xbe // Backlight Control 7
	CMD_BKLCTRL8 = 0xbf // Backlight Control 8

	CMD_PWCTRL1 = 0xc0 // Power Control 1
	CMD_PWCTRL2 = 0xc1 // Power Control 2
	CMD_VMCTRL1 = 0xc5 // VCOM Control 1
	CMD_VMCTRL2 = 0xc7 // VCOM Control 2
	CMD_PWCTRLA = 0xcb // Power Control A
	CMD_PWCTRLB = 0xcf // Power Control B

	CMD_NVMWR        = 0xd0 // NV Memory Write
	CMD_NVMPKEY      = 0xd1 // NV Memory Protection Key
	CMD_NVMSRD       = 0xd2 // NV Memory Status Read
	CMD_RDID4        = 0xd3 // Read ID4
	CMD_RDID1        = 0xda // Read ID1
	CMD_RDID2        = 0xdb // Read ID2
	CMD_RDID3        = 0xdc // Read ID3
	CMD_GAMCTRLP     = 0xe0 // Positive Gamma Control
	CMD_GAMCTRLN     = 0xe1 // Negative Gamma Control
	CMD_DGAMCTRL1    = 0xe2 // Ditigal Gamma Control 1
	CMD_DGAMCTRL2    = 0xe3 // Ditigal Gamma Control 2
	CMD_TIMCTRLA_INT = 0xe8 // Driver Timing Control A Internal
	CMD_TIMCTRLA_EXT = 0xe9 // Driver Timing Control A External
	CMD_TIMCTRLB     = 0xea // Driver Timing Control B
	CMD_PWSEQCTRL    = 0xed // Power on Sequence Control
	CMD_GAM3CTRL     = 0xf2 // Enable 3 Gamma Control
	CMD_IFCTRL       = 0xf6 // Interface Control
	CMD_PUMPRATIO    = 0xf7 // Pump Ratio Control
)

const (
	MADCTRL_MY  uint8 = 0x80 // Row Address Order         1 = address bottom to top
	MADCTRL_MX        = 0x40 // Column Address Order      1 = address right to left
	MADCTRL_MV        = 0x20 // Row/Column Exchange       1 = mirror and rotate 90 ccw
	MADCTRL_ML        = 0x10 // Vertical Refresh Order    1 = refresh bottom to top
	MADCTRL_BGR       = 0x08 // RGB-BGR Order             1 = Blue-Green-Red pixel order
	MADCTRL_MH        = 0x04 // Horizontal Refresh Order  1 = refresh right to left
)
