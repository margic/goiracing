package iracing

import (
	"encoding/binary"
)

// IRHeader represents the iracing memory mapped file header.
// This must be read to know where the offsets for the information about the which
// iracing variables are available.
type IRHeader struct {
	Ver      uint32 // api header version
	Status   uint32 // irsdk_StatusField
	TickRate uint32 // refresh rate being used by iracing to update the telemetry eg 60 or 360 Hz

	// Session info
	SessionInfoUpdate uint32 //increment on changes
	SessionInfoLen    uint32 // length in bytes of the session info yaml string
	SessionInfoOffset uint32 // location of the yaml data

	NumVars         uint32 // sounds like number of variables but may actually be the length of the variables data
	VarHeaderOffset uint32 // offset to the irsdk_varHeader[numVars] array

	NumBuf uint32
	BufLen uint32
	Pad1   [2]uint32

	bufInfos [4]IRBufInfo
}

const headerLengthBytes = 48 // number of bytes the iracing header consumes at start of mem mapped file

func NewIRHeader(b []byte) (*IRHeader, error) {
	h := &IRHeader{
		Ver:               binary.LittleEndian.Uint32(b[0:4]),
		Status:            binary.LittleEndian.Uint32(b[4:8]),
		TickRate:          binary.LittleEndian.Uint32(b[8:12]),
		SessionInfoUpdate: binary.LittleEndian.Uint32(b[12:16]),
		SessionInfoLen:    binary.LittleEndian.Uint32(b[16:20]),
		SessionInfoOffset: binary.LittleEndian.Uint32(b[20:24]),
		NumVars:           binary.LittleEndian.Uint32(b[24:28]),
		VarHeaderOffset:   binary.LittleEndian.Uint32(b[28:32]),
		NumBuf:            binary.LittleEndian.Uint32(b[32:36]),
		BufLen:            binary.LittleEndian.Uint32(b[36:40]),
	}
	h.Pad1[0] = binary.LittleEndian.Uint32(b[40:44])
	h.Pad1[1] = binary.LittleEndian.Uint32(b[44:48])

	return h, nil
}

type IRBufInfo struct {
	TickCount uint32
	BufOffset uint32
	Pad       [2]uint32
}
