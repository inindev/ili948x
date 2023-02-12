
set -e

BOARD=xiao-esp32c3
#BOARD=makerfabs-esp32c3spi35

TFT=tft_ili9341
#TFT=tft_ili948x

BIN=firmware.bin

MYFILE=$(readlink -f "$0")
MYDIR=$(dirname "${MYFILE}")

cd "$MYDIR"

rm -f $BIN
tinygo build -tags $TFT -target=$BOARD -o $BIN .
esptool.py -p /dev/cu.usbmodem* -b 921600 write_flash 0 $BIN

#tinygo monitor -port /dev/cu.usbserial*
#screen /dev/cu.usbserial* 115200

