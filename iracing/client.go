package iracing

import (
	"encoding/json"
	"fmt"
	"strings"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"
)

const iracingMemoryMappedFileName string = "Local\\IRSDKMemMapFileName"

type Client struct {
	logger *zap.Logger
	ptr    uintptr // unsafe pointer to iracing mem mapped file
	// headerSlice Mmap    // a slice that represents the ir header in the mem mapped file
	header          *Header
	sessionInfoYaml string
}

func NewClient() *Client {
	logger := newLogger()
	return &Client{
		logger: logger,
	}
}

func (ir *Client) Close() error {
	ir.logger.Sync()
	return nil
}

// Use was taken from syscall package:
// Use is a no-op, but the compiler cannot see that it is.
// Calling Use(p) ensures that p is kept live until that point.
func use(unsafe.Pointer) {}

func (ir *Client) Open() error {
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
	ir.ptr = addr

	// set up a slice using the windows pointer to mem map file
	headerSlice := Mmap{}
	h := headerSlice.Header()
	h.Data = ir.ptr
	h.Cap = headerLengthBytes
	h.Len = headerLengthBytes

	// create new header to parse bytes
	ir.header, err = NewHeader(headerSlice)
	if err != nil {
		return err
	}

	ir.logger.Debug("iracing header",
		zap.Uint32("version", ir.header.Ver),
		zap.Uint32("status", ir.header.Status),
		zap.Uint32("tickrate", ir.header.TickRate),
		zap.Uint32("sessionInfoUpdate", ir.header.SessionInfoUpdate),
		zap.Uint32("infolength", ir.header.SessionInfoLen),
		zap.Uint32("infoOffset", ir.header.SessionInfoOffset),
		zap.Uint32("numVars", ir.header.NumVars),
		zap.Uint32("varOffset", ir.header.VarHeaderOffset),
		zap.Uint32("numBuf", ir.header.NumBuf),
		zap.Uint32("BufLen", ir.header.BufLen))

	ir.sessionInfoYaml = ir.readSession()
	ir.logger.Debug("session info string", zap.String("session", ir.sessionInfoYaml))
	return nil
}

func (ir *Client) readSession() string {
	// setup a slice around the area of the file with the session data in it
	// we'll find a more efficient way to deal with this once its working
	sessionInfoSlice := Mmap{}
	// set the slice header to make this slice point to our data
	h := sessionInfoSlice.Header()
	h.Data = ir.ptr + uintptr(ir.header.SessionInfoOffset)
	h.Cap = int(ir.header.SessionInfoLen)
	h.Len = int(ir.header.SessionInfoLen)

	// h.Cap = int(ir.irHeader.SessionInfoLen + ir.irHeader.SessionInfoOffset)
	// h.Len = int(ir.irHeader.SessionInfoLen + ir.irHeader.SessionInfoOffset)

	// TODO Working here trying to read the session info data. Having issues with the size of the info data
	infoStr := string(sessionInfoSlice)
	h.Len = strings.LastIndex(infoStr, "...")
	infoStr = string(sessionInfoSlice)
	return fmt.Sprintf("irSessionInfoSlice %s", infoStr)
}

func newLogger() *zap.Logger {
	rawJSON := []byte(`{
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"initialFields": {},
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger
}
