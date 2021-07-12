package iracing

import (
	"fmt"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"
)

const iracingMemoryMappedFileName string = "Local\\IRSDKMemMapFileName"

type IRClient struct {
	logger *zap.Logger
	irPtr  uintptr // unsafe pointer to iracing mem mapped file
	// headerSlice Mmap    // a slice that represents the ir header in the mem mapped file
	irHeader          *IRHeader
	irSessionInfoYaml string
}

func NewIRClient(logger *zap.Logger) *IRClient {
	return &IRClient{
		logger: logger,
	}
}

func (ir *IRClient) Close() error {
	return nil
}

// Use was taken from syscall package:
// Use is a no-op, but the compiler cannot see that it is.
// Calling Use(p) ensures that p is kept live until that point.
func use(unsafe.Pointer) {}

func (ir *IRClient) Open() error {
	ir.logger.Debug("opening iracing memory mapped file", zap.String("filename", iracingMemoryMappedFileName))
	ptrName, err := windows.UTF16PtrFromString(iracingMemoryMappedFileName)
	if err != nil {
		return err
	}
	uPtrName := unsafe.Pointer(ptrName)

	modkernel32 := windows.NewLazyDLL("kernel32.dll")
	procOpenFileMapping := modkernel32.NewProc("OpenFileMappingW")
	winHandle, _, err := procOpenFileMapping.Call(uintptr(4), uintptr(0), uintptr(uPtrName))
	if winHandle == 0 && err != nil {
		return err
	}
	use(uPtrName) // see use

	addr, err := windows.MapViewOfFile(windows.Handle(winHandle), uint32(windows.FILE_MAP_READ), 0, 0, 0)
	if err != nil {
		return err
	}
	ir.irPtr = addr

	// set up a slice using the windows pointer to mem map file
	irHeaderSlice := Mmap{}
	h := irHeaderSlice.Header()
	h.Data = ir.irPtr
	h.Cap = headerLengthBytes
	h.Len = headerLengthBytes

	// create new header to parse bytes
	ir.irHeader, err = NewIRHeader(irHeaderSlice)
	if err != nil {
		return err
	}

	ir.logger.Debug("iracing header",
		zap.Uint32("version", ir.irHeader.Ver),
		zap.Uint32("status", ir.irHeader.Status),
		zap.Uint32("tickrate", ir.irHeader.TickRate),
		zap.Uint32("sessionInfoUpdate", ir.irHeader.SessionInfoUpdate),
		zap.Uint32("infolength", ir.irHeader.SessionInfoLen),
		zap.Uint32("infoOffset", ir.irHeader.SessionInfoOffset),
		zap.Uint32("numVars", ir.irHeader.NumVars),
		zap.Uint32("varOffset", ir.irHeader.VarHeaderOffset),
		zap.Uint32("numBuf", ir.irHeader.NumBuf),
		zap.Uint32("BufLen", ir.irHeader.BufLen))

	ir.irSessionInfoYaml = ir.readSession()
	ir.logger.Debug("session info string", zap.String("session", ir.irSessionInfoYaml))
	return nil
}

func (ir *IRClient) readSession() string {
	// setup a slice around the area of the file with the session data in it
	// we'll find a more efficient way to deal with this once its working
	irSessionInfoSlice := Mmap{}
	// set the slice header to make this slice point to our data
	h := irSessionInfoSlice.Header()
	h.Data = ir.irPtr + uintptr(ir.irHeader.SessionInfoOffset) + 4
	h.Cap = int(ir.irHeader.SessionInfoLen - 1)
	h.Len = int(ir.irHeader.SessionInfoLen - 1)

	// h.Cap = int(ir.irHeader.SessionInfoLen + ir.irHeader.SessionInfoOffset)
	// h.Len = int(ir.irHeader.SessionInfoLen + ir.irHeader.SessionInfoOffset)

	// TODO Working here trying to read the session info data. Having issues with the size of the info data
	return fmt.Sprintf("irSessionInfoSlice %d", len(irSessionInfoSlice))
}
