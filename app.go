package main

import (
	"machine"
	"os"
	"time"

	"tinygo.org/x/drivers/sdcard"
	"tinygo.org/x/tinyfs/fatfs"
)

const (
	TFT_SCK_PIN = machine.SPI_SCK_PIN
	TFT_SDO_PIN = machine.SPI_SDO_PIN
	TFT_SDI_PIN = machine.SPI_SDI_PIN
	TFT_CS_PIN  = machine.GPIO6
	TFT_DC_PIN  = machine.GPIO7
	TFT_BL_PIN  = machine.NoPin
	TFT_RST_PIN = machine.NoPin

	SD_SCK_PIN = machine.NoPin
	SD_SDO_PIN = machine.NoPin
	SD_SDI_PIN = machine.NoPin
	SD_CS_PIN  = machine.NoPin

	/*
		TFT_SCK_PIN = machine.TFT_SCK_PIN
		TFT_SDO_PIN = machine.TFT_SDO_PIN
		TFT_SDI_PIN = machine.TFT_SDI_PIN
		TFT_CS_PIN = machine.TFT_CS_PIN
		TFT_DC_PIN = machine.TFT_DC_PIN
		TFT_BL_PIN = machine.TFT_BL_PIN
		TFT_RST_PIN = machine.NoPin

		SD_SCK_PIN = machine.SD_SCK_PIN
		SD_SDO_PIN = machine.SD_SDO_PIN
		SD_SDI_PIN = machine.SD_SDI_PIN
		SD_CS_PIN = machine.SD_CS_PIN
	*/
)

// https://www.w3schools.com/colors/colors_wheels.asp
const (
	BLACK uint32 = 0x000000
	WHITE uint32 = 0xffffff

	// https://www.w3schools.com/colors/pic_cmyk_wheel.gif
	CMY_CYAN    uint32 = 0x00ffff
	CMY_CBLUE   uint32 = 0x0080ff
	CMY_BLUE    uint32 = 0x0000ff
	CMY_PURPLE  uint32 = 0x8000ff
	CMY_MAGENTA uint32 = 0xff00ff
	CMY_MRED    uint32 = 0xff0080
	CMY_RED     uint32 = 0xff0000
	CMY_ORANGE  uint32 = 0xff8000
	CMY_YELLOW  uint32 = 0xffff00
	CMY_YGREEN  uint32 = 0x80ff00
	CMY_GREEN   uint32 = 0x00ff00
	CMY_CGREEN  uint32 = 0x00ff80

	// https://www.w3schools.com/colors/pic_ryb_itten.jpg
	RYB_RED     uint32 = 0xfe2712
	RYB_RORANGE uint32 = 0xfc600a
	RYB_ORANGE  uint32 = 0xfb9902
	RYB_YORANGE uint32 = 0xfccc1a
	RYB_YELLOW  uint32 = 0xfefe33
	RYB_YGREEN  uint32 = 0xb2d732
	RYB_GREEN   uint32 = 0x66b032
	RYB_BGREEN  uint32 = 0x347c98
	RYB_BLUE    uint32 = 0x0247fe
	RYB_BPURPLE uint32 = 0x4424d6
	RYB_PURPLE  uint32 = 0x8601af
	RYB_RPURPLE uint32 = 0xc21460
)

var (
	cmy_colors []uint32 = []uint32{CMY_CYAN, CMY_CBLUE, CMY_BLUE, CMY_PURPLE, CMY_MAGENTA, CMY_MRED, CMY_RED, CMY_ORANGE, CMY_YELLOW, CMY_YGREEN, CMY_GREEN, CMY_CGREEN}
	ryb_colors []uint32 = []uint32{RYB_RED, RYB_RORANGE, RYB_ORANGE, RYB_YORANGE, RYB_YELLOW, RYB_YGREEN, RYB_GREEN, RYB_BGREEN, RYB_BLUE, RYB_BPURPLE, RYB_PURPLE, RYB_RPURPLE}
)

func main() {
	machine.SPI2.Configure(machine.SPIConfig{
		SCK: TFT_SCK_PIN,
		SDO: TFT_SDO_PIN,
		SDI: TFT_SDI_PIN,
		//CS:        TFT_CS_PIN,
		LSBFirst:  false,
		Mode:      machine.SPI_MODE0,
		Frequency: 40e6,
	})

	tft := NewTFTPanel(
		NewSPITransport(machine.SPI2),
		TFT_CS_PIN,  // chip select
		TFT_DC_PIN,  // data / command
		TFT_BL_PIN,  // backlight
		TFT_RST_PIN, // reset
		TFT_DEFAULT_WIDTH,
		TFT_DEFAULT_HEIGHT)

	//tft.screenFillDemo(ryb_colors)
	tft.screenFillDemo(cmy_colors)

	//tft.quadrantDemo()
	tft.rotateDemo(tft.quadrantDemo, 1500, 4)

	//tft.colorBlocksDemo()
	tft.rotateDemo(tft.colorBlocksDemo, 500, 8)

	//tft.stackedRectanglesDemo()
	tft.rotateDemo(tft.stackedRectanglesDemo, 1500, 6)

	/*
		tft.SetRotation(Rot_90)
		tft.bitmapDemo("/logo.bmp")
		time.Sleep(time.Second)

		tft.SetRotation(Rot_270)
		tft.bitmapDemo("/logo.bmp")

		// scroll demo
		tfa := uint16(15)
		bfa := uint16(160)
		tft.SetScrollArea(tfa, bfa)

		for i := tfa + 1; i < 480-bfa; i++ {
			tft.SetScroll(i)
			time.Sleep(time.Millisecond * 25)
		}
		time.Sleep(time.Second)

		for i := 480 - bfa - 1; i >= tfa; i-- {
			tft.SetScroll(i)
			time.Sleep(time.Millisecond * 25)
		}
		time.Sleep(time.Second)

		tft.StopScroll()
	*/
	for {
		time.Sleep(time.Minute)
	}
}

func (tft *TFTPanel) screenFillDemo(palette []uint32) {
	for _, color := range palette {
		tft.FillScreen(color)
		time.Sleep(time.Millisecond * 1000)
	}
}

func (tft *TFTPanel) quadrantDemo() {
	cfa := []uint32{RYB_BGREEN, RYB_BPURPLE}
	cba := []uint32{RYB_YORANGE, RYB_YGREEN}

	i := uint8(tft.GetRotation())
	tft.FillScreen(cba[i%2])
	tft.FillRectangle(10, 10, 50, 50, cfa[i%2])
	width, height := tft.Size()
	tft.DrawHLine(10, width-20, height/3, cfa[i%2])
	tft.DrawVLine(width/2, 10, height-20, cfa[i%2])
}

func (tft *TFTPanel) colorBlocksDemo() {
	palette := [10][10]uint32{
		{0xfdedec, 0xfadbd8, 0xf5b7b1, 0xf1948a, 0xec7063, 0xe74c3c, 0xcb4335, 0xb03a2e, 0x943126, 0x78281f}, // reds
		{0xf4ecf7, 0xe8daef, 0xd2b4de, 0xbb8fce, 0xa569bd, 0x8e44ad, 0x7d3c98, 0x6c3483, 0x5b2c6f, 0x4a235a}, // purples
		{0xebf5fb, 0xd6eaf8, 0xaed6f1, 0x85c1e9, 0x5dade2, 0x3498db, 0x2e86c1, 0x2874a6, 0x21618c, 0x1b4f72}, // blues
		{0xe8f6f3, 0xd0ece7, 0xa2d9ce, 0x73c6b6, 0x45b39d, 0x16a085, 0x138d75, 0x117a65, 0x0e6655, 0x0b5345}, // d-greens
		{0xeafaf1, 0xd5f5e3, 0xabebc6, 0x82e0aa, 0x58d68d, 0x2ecc71, 0x28b463, 0x239b56, 0x1d8348, 0x186a3b}, // l-greens
		{0xfef9e7, 0xfcf3cf, 0xf9e79f, 0xf7dc6f, 0xf4d03f, 0xf1c40f, 0xd4ac0d, 0xb7950b, 0x9a7d0a, 0x7d6608}, // yellows
		{0xfef5e7, 0xfdebd0, 0xfad7a0, 0xf8c471, 0xf5b041, 0xf39c12, 0xd68910, 0xb9770e, 0x9c640c, 0x7e5109}, // oranges
		{0xfbeee6, 0xf6ddcc, 0xedbb99, 0xe59866, 0xdc7633, 0xd35400, 0xba4a00, 0xa04000, 0x873600, 0x6e2c00}, // browns
		{0xf8f9f9, 0xf2f3f4, 0xe5e7e9, 0xd7dbdd, 0xcacfd2, 0xbdc3c7, 0xa6acaf, 0x909497, 0x797d7f, 0x626567}, // grays
		{0xeaecee, 0xd5d8dc, 0xabb2b9, 0x808b96, 0x566573, 0x2c3e50, 0x273746, 0x212f3d, 0x1c2833, 0x17202a}, // steel
	}

	width, height := tft.Size()
	bw := width / uint16(10)
	bh := height / uint16(10)
	for x := uint16(0); x < 10; x++ {
		for y := uint16(0); y < 10; y++ {
			tft.FillRectangle(x*bw, y*bh, bw, bh, palette[x][y])
		}
	}
}

func (tft *TFTPanel) stackedRectanglesDemo() {
	const (
		G_RED uint32 = 0xea4335

		CUL = CMY_BLUE
		CUR = G_RED
		CLL = RYB_GREEN
		CLR = RYB_YORANGE
		CMT = CMY_ORANGE
	)

	width, height := tft.Size()
	tft.FillRectangle(0, 0, width/2, height/2, CUL)              // upper-left
	tft.FillRectangle(width/2, 0, width/2, height/2, CUR)        // upper-right
	tft.FillRectangle(0, height/2, width/2, height/2, CLL)       // lower-left
	tft.FillRectangle(width/2, height/2, width/2, height/2, CLR) // lower-right
	tft.FillRectangle(width/4, height/4, width/2, height/2, CMT) // middle
}

func (tft *TFTPanel) bitmapDemo(filename string) {
	sd := sdcard.New(&machine.SPI2,
		SD_SCK_PIN,
		SD_SDO_PIN,
		SD_SDI_PIN,
		SD_CS_PIN)
	err := sd.Configure()
	if err != nil {
		printError("failed to bind sdcard device", "", err)
		return
	}

	filesystem := fatfs.New(&sd)
	filesystem.Configure(&fatfs.Config{
		SectorSize: 512,
	})

	f, err := filesystem.OpenFile(filename, os.O_RDONLY)
	if err != nil {
		printError("could not open file", filename, err)
		return
	}
	defer f.Close()

	// https://en.wikipedia.org/wiki/BMP_file_format
	header := make([]byte, 14)
	f.Read(header)
	//print(hex.Dump(header))
	if header[0] != 'B' || header[1] != 'M' {
		printError("bitmap file is invalid", "", err)
		return
	}
	img_offs := uint32(header[10]) | uint32(header[11])<<8 | uint32(header[12])<<16 | uint32(header[13])<<24

	// reuse buffer to read past remaining header
	q := img_offs / uint32(len(header))
	for i := uint32(1); i < q; i++ {
		f.Read(header)
	}
	r := img_offs % uint32(len(header))
	f.Read(header[:r])

	width, height := tft.Size()
	tft.DisplayBitmap(0, 0, width, height, 24, f)
}

func (tft *TFTPanel) rotateDemo(pfunc func(), delayMs time.Duration, count int) {
	for i := 0; i < count; i++ {
		tft.SetRotation(Rotation(i % 4))
		pfunc()
		time.Sleep(time.Millisecond * delayMs)
	}
}

func printError(msg, key string, err error) {
	print(msg)
	if key != "" {
		print(" ")
		print(key)
	}
	print(" - error: ")
	print(err.Error())
	print("\r\n")
}
