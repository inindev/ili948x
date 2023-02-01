package main

import (
	"machine"
)

type spiTransport struct {
	spi machine.SPI // spi bus
	buf []uint8     // spi data buffer
}

func NewSPITransport(spi machine.SPI) iTransport {
	return &spiTransport{
		spi: spi,
		buf: make([]uint8, 64),
	}
}

// 8 bit
func (st *spiTransport) write8(data uint8) {
	st.buf[0] = data
	st.spi.Tx(st.buf[:1], nil)
}

func (st *spiTransport) write8n(data uint8, n int) {
	writeNn[uint8](st, data, n, 1)
}

func (st *spiTransport) write8sl(data []uint8) {
	writeNsl[uint8](st, data, 1)
}

// 16 bit
func (st *spiTransport) write16(data uint16) {
	st.buf[0] = uint8(data)
	st.buf[1] = uint8(data >> 8)
	st.spi.Tx(st.buf[:2], nil)
}

func (st *spiTransport) write16n(data uint16, n int) {
	writeNn[uint16](st, data, n, 2)
}

func (st *spiTransport) write16sl(data []uint16) {
	writeNsl[uint16](st, data, 2)
}

// 24 bit
func (st *spiTransport) write24(data uint32) {
	st.buf[0] = uint8(data)
	st.buf[1] = uint8(data >> 8)
	st.buf[2] = uint8(data >> 16)
	st.spi.Tx(st.buf[:3], nil)
}

func (st *spiTransport) write24n(data uint32, n int) {
	writeNn[uint32](st, data, n, 3)
}

func (st *spiTransport) write24sl(data []uint32) {
	writeNsl[uint32](st, data, 3)
}

func writeNn[T int8 | uint8 | int16 | uint16 | int32 | uint32](st *spiTransport, data T, n, bytes int) {
	dataBytes := n * bytes
	bufBytes := (len(st.buf) / bytes) * bytes

	for i := 0; i < n; i++ {
		pos := (i * bytes) % bufBytes
		for j := 0; j < bytes; j++ {
			st.buf[pos] = uint8(data >> (j * 8))
			pos++
		}
		if pos >= bufBytes || pos >= dataBytes {
			st.spi.Tx(st.buf[:pos], nil) // transmit
			dataBytes -= pos
		}
	}
}

func writeNsl[T int8 | uint8 | int16 | uint16 | int32 | uint32](st *spiTransport, data []T, bytes int) {
	dataBytes := len(data) * bytes
	bufBytes := (len(st.buf) / bytes) * bytes

	for i, elem := range data {
		pos := (i * bytes) % bufBytes
		for j := 0; j < bytes; j++ {
			st.buf[pos] = uint8(elem >> (j * 8))
			pos++
		}
		if pos >= bufBytes || pos >= dataBytes {
			st.spi.Tx(st.buf[:pos], nil) // transmit
			dataBytes -= pos
		}
	}
}
