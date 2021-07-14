package iracing

import (
	"encoding/binary"

	"golang.org/x/sys/windows"
)

const varHeaderLenth int = 144

// varHeader represents the variable header in iracing telemtry
// The variable header represets a variable length telemetry variable stored at the offset
// this allows us to know where in the telemetry buffer to read the values and how
// many values there are. The length to read at offset is determined by count * type
type varHeader struct {
	t           varType
	offset      int
	count       int
	countAsTime bool
	// pad         [3]byte there is padding in the iracing record but we will ignore
	name string // max 32
	desc string // max 64
	unit string // max 32
}

type varTypeLength int

const (
	ircharLen     varTypeLength = 1
	irboolLen     varTypeLength = 1
	irintLen      varTypeLength = 4
	irbitFieldLen varTypeLength = 4
	irfloatLen    varTypeLength = 4
	irdoubleLen   varTypeLength = 8
)

func (w varTypeLength) EnumIndex() int {
	return int(w)
}

type varType int

const (
	// 1 byte
	irchar varType = iota
	irbool

	// 4 bytes
	irint
	irbitField
	irfloat

	// 8 bytes
	irdouble
)

func (t varType) String() string {
	return [...]string{"char", "bool", "int", "bitField", "float", "double"}[t]
}

func (w varType) EnumIndex() int {
	return int(w)
}

// newVarHeader takes a byte slice and uses binary endoding to read the values and populate the header
func newVarHeader(b []byte) *varHeader {
	h := &varHeader{
		t:           varType(binary.LittleEndian.Uint32(b[0:4])),
		offset:      int(binary.LittleEndian.Uint32(b[4:8])),
		count:       int(binary.LittleEndian.Uint32(b[8:12])),
		countAsTime: b[12] != 0,
		// pad         [3]byte there is padding in the iracing record but we will ignore
		name: windows.ByteSliceToString(b[16:48]),
		desc: windows.ByteSliceToString(b[48:112]),
		unit: windows.ByteSliceToString(b[112:144]),
	}
	return h
}
