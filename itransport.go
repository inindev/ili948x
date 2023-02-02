package main

type iTransport interface {
	// 8 bit
	write8(b uint8)
	write8n(b uint8, n int)
	write8sl(b []uint8)

	// 16 bit
	write16(data uint16)
	write16n(data uint16, n int)
	write16sl(data []uint16)

	// 24 bit
	write24(data uint32)
	write24n(data uint32, n int)
	write24sl(data []uint32)
}
