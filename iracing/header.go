package iracing

import (
	"encoding/binary"
)

// IRHeader represents the iracing memory mapped file header.
// This must be read to know where the offsets for the information about the which
// iracing variables are available.
type IRHeader struct {
	Ver      int // api header version
	Status   int // irsdk_StatusField
	TickRate int // refresh rate being used by iracing to update the telemetry eg 60 or 360 Hz

	// Session info
	SessionInfoTickCount int //increment on changes
	SessionInfoLen       int // length in bytes of the session info yaml string
	SessionInfoOffset    int // location of the yaml data

	NumVars         int // sounds like number of variables but may actually be the length of the variables data
	VarHeaderOffset int // offset to the irsdk_varHeader[numVars] array

	NumBuf int
	BufLen int
	// Pad1   [2]uint32 // there is padding in the iracing sdk but no need in this client

	bufInfos [4]BufInfo
}

const headerLength = 48 // number of bytes the iracing header consumes at start of mem mapped file

func newHeader(b []byte) *IRHeader {
	h := &IRHeader{
		Ver:                  int(binary.LittleEndian.Uint32(b[0:4])),
		Status:               int(binary.LittleEndian.Uint32(b[4:8])),
		TickRate:             int(binary.LittleEndian.Uint32(b[8:12])),
		SessionInfoTickCount: int(binary.LittleEndian.Uint32(b[12:16])),
		SessionInfoLen:       int(binary.LittleEndian.Uint32(b[16:20])),
		SessionInfoOffset:    int(binary.LittleEndian.Uint32(b[20:24])),
		NumVars:              int(binary.LittleEndian.Uint32(b[24:28])),
		VarHeaderOffset:      int(binary.LittleEndian.Uint32(b[28:32])),
		NumBuf:               int(binary.LittleEndian.Uint32(b[32:36])),
		BufLen:               int(binary.LittleEndian.Uint32(b[36:40])),
	}
	// h.Pad1[0] = binary.LittleEndian.Uint32(b[40:44])
	// h.Pad1[1] = binary.LittleEndian.Uint32(b[44:48])

	return h
}

type BufInfo struct {
	TickCount int
	BufOffset int
	Pad       [2]int
}
