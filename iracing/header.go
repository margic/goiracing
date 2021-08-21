package iracing

import (
	"encoding/binary"
	"fmt"
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

	// iracing sdk implies 4 current buffers verify if this - num
	BufInfos []*BufInfo
}

const headerLength = 40 // number of bytes the iracing header consumes at start of mem mapped file

func newHeader(irMemMapedFile uintptr) *IRHeader {

	// set up a slice using the windows pointer to mem map file
	headerSlice := Mmap{}
	h := headerSlice.Header()
	h.Data = irMemMapedFile
	h.Cap = headerLength
	h.Len = headerLength

	header := &IRHeader{
		Ver:                  int(binary.LittleEndian.Uint32(headerSlice[0:4])),
		Status:               int(binary.LittleEndian.Uint32(headerSlice[4:8])),
		TickRate:             int(binary.LittleEndian.Uint32(headerSlice[8:12])),
		SessionInfoTickCount: int(binary.LittleEndian.Uint32(headerSlice[12:16])),
		SessionInfoLen:       int(binary.LittleEndian.Uint32(headerSlice[16:20])),
		SessionInfoOffset:    int(binary.LittleEndian.Uint32(headerSlice[20:24])),
		NumVars:              int(binary.LittleEndian.Uint32(headerSlice[24:28])),
		VarHeaderOffset:      int(binary.LittleEndian.Uint32(headerSlice[28:32])),
		NumBuf:               int(binary.LittleEndian.Uint32(headerSlice[32:36])),
		BufLen:               int(binary.LittleEndian.Uint32(headerSlice[36:40])),
	}

	bufInfoSlice := Mmap{}
	h = bufInfoSlice.Header()
	h.Data = irMemMapedFile + 48
	h.Cap = header.NumBuf * header.BufLen
	h.Len = header.NumBuf * header.BufLen

	fmt.Printf("bufCount %d\r\n", header.NumBuf)
	bufInfos := make([]*BufInfo, header.NumBuf)
	infoLen := 16
	for i := 0; i < header.NumBuf; i++ {
		s := i * infoLen
		bufInfo := &BufInfo{
			TickCount: int(binary.LittleEndian.Uint32(bufInfoSlice[s : s+4])),
			BufOffset: int(binary.LittleEndian.Uint32(bufInfoSlice[s+4 : s+8])),
		}
		bufInfos[i] = bufInfo
	}
	header.BufInfos = bufInfos
	fmt.Printf("bufInfos %d\r\n", len(bufInfos))
	return header
}

type BufInfo struct {
	TickCount int
	BufOffset int
}
