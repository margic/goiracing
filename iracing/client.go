package iracing

import (
	"encoding/json"
	"fmt"
	"os"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"
)

const iracingMemoryMappedFileName string = "Local\\IRSDKMemMapFileName"
const (
	success = iota
	fileError
	readError
)

type Client struct {
	logger *zap.Logger
	ptr    uintptr // unsafe pointer to iracing mem mapped file
	// headerSlice Mmap    // a slice that represents the ir header in the mem mapped file
	header          *header
	varHeaders      map[string]*varHeader // I think this may change frequently depends on if offsets are static consider a lock
	SessionInfoYaml string
}

type ClientConfig struct {
	Debug bool
}

func NewClient(cfg *ClientConfig) *Client {
	logger := newLogger(cfg.Debug)
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

func (ir *Client) Open() {
	ir.logger.Debug("opening iracing memory mapped file", zap.String("filename", iracingMemoryMappedFileName))
	ptrName, err := windows.UTF16PtrFromString(iracingMemoryMappedFileName)
	if err != nil {
		ir.logger.Error("Error creating windows pointer from file name",
			zap.String("filename", iracingMemoryMappedFileName),
			zap.Error(err))
		os.Exit(fileError)
	}
	uPtrName := unsafe.Pointer(ptrName)

	modkernel32 := windows.NewLazyDLL("kernel32.dll")
	procOpenFileMapping := modkernel32.NewProc("OpenFileMappingW")
	winHandle, _, err := procOpenFileMapping.Call(uintptr(4), uintptr(0), uintptr(uPtrName))
	if winHandle == 0 && err != nil {
		ir.logger.Error("Error opening windows memory mapped file",
			zap.String("filename", iracingMemoryMappedFileName),
			zap.Error(err))
		os.Exit(fileError)
	}
	use(uPtrName) // see use

	addr, err := windows.MapViewOfFile(windows.Handle(winHandle), uint32(windows.FILE_MAP_READ), 0, 0, 0)
	if err != nil {
		ir.logger.Error("Error creating map view of file",
			zap.String("filename", iracingMemoryMappedFileName),
			zap.Error(err))
		os.Exit(fileError)
	}
	ir.ptr = addr

	// set up a slice using the windows pointer to mem map file
	headerSlice := Mmap{}
	h := headerSlice.Header()
	h.Data = ir.ptr
	h.Cap = headerLength
	h.Len = headerLength

	// create new header to parse bytes
	ir.header = newHeader(headerSlice)

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

	ir.SessionInfoYaml = ir.readSession()
	ir.logger.Debug("session info string", zap.String("session", ir.SessionInfoYaml))
	ir.readVariableHeaders()
}

func (ir *Client) readVariableHeaders() {
	// setup a slice around the variable headers data in the iracing ptr based on offet and variable header length
	varHeaderSlice := Mmap{}
	h := varHeaderSlice.Header()
	h.Data = ir.ptr + uintptr(ir.header.VarHeaderOffset)
	h.Cap = varHeaderLenth * int(ir.header.NumVars)
	h.Len = varHeaderLenth * int(ir.header.NumVars)

	// initialize a map to store telementry variable headers mapped by name
	varHeaders := make(map[string]*varHeader)

	// loop through the variable headers byte slice to read ir.header.NumVars headers and create a variable header to add to map
	for i := 0; i < int(ir.header.NumVars); i++ {
		b := varHeaderSlice[i*varHeaderLenth : (i+1)*varHeaderLenth]
		h := newVarHeader(b)
		varHeaders[h.name] = h
		ir.logger.Debug("variable header",
			zap.String("name", h.name),
			zap.String("description", h.desc),
			zap.String("type", h.t.String()),
		)
	}
	ir.logger.Debug("parsed variable headers", zap.Int("numvars", int(ir.header.NumVars)))
	ir.varHeaders = varHeaders
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
	infoStr := windows.ByteSliceToString(sessionInfoSlice)
	return infoStr
}

func newLogger(debug bool) *zap.Logger {
	var level string
	if debug {
		level = "debug"
	} else {
		level = "info"
	}
	cfgStr := `{
		"level": "%s",
		"encoding": "json",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"initialFields": {},
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`

	rawJSON := []byte(fmt.Sprintf(cfgStr, level))

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
