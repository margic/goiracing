package iracing

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"
)

const iracingMemoryMappedFileName string = "Local\\IRSDKMemMapFileName"
const (
	closed = iota
	open
	loadedHeader
	loadedVarHeaders
	loadedVarBuf
)

type Client struct {
	logger         *zap.Logger
	irMemMapedFile uintptr // unsafe pointer to iracing mem mapped file
	// headerSlice Mmap    // a slice that represents the ir header in the mem mapped file
	header               *IRHeader
	varHeaders           map[string]*varHeader // I think this may change frequently depends on if offsets are static consider a lock
	SessionInfoYaml      string
	stop                 bool
	retryInterval        int
	sessionInfoTickCount int
	varBufTickCount      int
	varBuf               []byte
	varBufLock           sync.Mutex
	status               int
}

type ClientConfig struct {
	Debug         bool
	RetryInterval int
}

func (ir *Client) Emit(varName string) {
	// TODO change how we open the file so it's retry is within scope of graceful shutdown
	err := ir.open()
	if err != nil {
		ir.logger.Error("error opening client", zap.Error(err))
		return
	}

	// pre read header
	err = ir.readHeader()
	if err != nil {
		ir.logger.Error("error reading iracing header", zap.Error(err))
	}

	// pre read var headers
	err = ir.readVarHeaders()
	if err != nil {
		ir.logger.Error("error reading variable headers", zap.Error(err))
	}

	// Start background tasks to read variables bufs and keep them refreshed
	// also start variable read background tasks to read

	// setup signal channels to run reads in background
	sigReadVarBuf := make(chan bool, 1)

	// start ticker to invoke variable reads
	ticker := time.NewTicker(20 * time.Millisecond)
	go func() {
		for {
			t := <-ticker.C
			ir.logger.Debug("TEMP debug. Ticker ticked read varBufs", zap.Time("tick", t))
			err := ir.readVarBuf()
			if err != nil {
				ir.logger.Error("error reading variable buffer", zap.Error(err))
			}
			sigReadVarBuf <- true
		}
	}()

	go func() {
		// setup output
		o := &Output{}
		out := o.OutputChannel()
		for {
			<-sigReadVarBuf
			s := &Suspension{
				LFShockDef: ir.readFloat32Var("LFshockDef"),
				LFShockVel: ir.readFloat32Var("LFshockVel"),
				RFShockDef: ir.readFloat32Var("RFshockDef"),
				RFShockVel: ir.readFloat32Var("RFshockVel"),
				LRShockDef: ir.readFloat32Var("LRshockDef"),
				LRShockVel: ir.readFloat32Var("LRshockVel"),
				RRShockDef: ir.readFloat32Var("RRshockDef"),
				RRShockVel: ir.readFloat32Var("RRshockVel"),
			}
			out <- s
		}
	}()

	// graceful shutdown figure this out once it works
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ticker.Stop()

	time.Sleep(100 * time.Millisecond)

	ir.close()
}

func (ir *Client) Session() {
	err := ir.open()
	if err != nil {
		ir.logger.Error("error opening client", zap.Error(err))
		return
	}
	defer ir.close()

	err = ir.readHeader()
	if err != nil {
		ir.logger.Error("error reading iracing header", zap.Error(err))
	}
	fmt.Printf(ir.SessionInfoYaml)
}

func (ir *Client) Variables() {
	err := ir.open()
	if err != nil {
		ir.logger.Error("error opening client", zap.Error(err))
		return
	}
	defer ir.close()

	err = ir.readHeader()
	if err != nil {
		ir.logger.Error("error reading iracing header", zap.Error(err))
	}

	err = ir.readVarHeaders()
	if err != nil {
		ir.logger.Error("error reading variable headers", zap.Error(err))
	}

	for name, header := range ir.varHeaders {
		ir.logger.Debug("varHeader", zap.String("name", name), zap.String("desc", header.desc), zap.Int("offset", header.offset))
	}

	err = ir.readVarBuf()
	if err != nil {
		ir.logger.Error("error reading variable buffer", zap.Error(err))
	}

	// err = ir.readVar()
	// if err != nil {
	// 	ir.logger.Error("error reading variable", zap.Error(err))
	// }
}

func NewClient(cfg *ClientConfig) *Client {
	logger := newLogger(cfg.Debug)
	c := &Client{
		logger: logger,
	}
	if cfg.RetryInterval > 0 {
		c.retryInterval = cfg.RetryInterval
	} else {
		c.retryInterval = 10
	}
	c.status = closed
	c.varBufTickCount = 0
	return c
}

func (ir *Client) close() {
	ir.logger.Debug("closing iracing client")
	ir.logger.Sync()
	ir.stop = true
	ir.status = closed
}

// Use was taken from syscall package:
// Use is a no-op, but the compiler cannot see that it is.
// Calling Use(p) ensures that p is kept live until that point.
func (ir *Client) use(unsafe.Pointer) {}

func (ir *Client) open() error {
	if ir.status != closed {
		return fmt.Errorf("invalid client status for open iracing mem mapped file status %d", ir.status)
	}
	winHandle, err := ir.openIracingFile()
	if err != nil {
		return err
	}
	ir.logger.Debug("opening map view of file")
	addr, err := windows.MapViewOfFile(windows.Handle(winHandle), uint32(windows.FILE_MAP_READ), 0, 0, 0)
	if err != nil {
		ir.logger.Error("Error creating map view of file",
			zap.String("filename", iracingMemoryMappedFileName),
			zap.Error(err))
		return err
	}
	ir.logger.Debug("got map view of file", zap.Uintptr("address", addr))
	ir.irMemMapedFile = addr
	return nil
}

// openIracingFile will loop and wait for an iracing file to exist.
func (ir *Client) openIracingFile() (uintptr, error) {
	if ir.status != closed {
		return 0, fmt.Errorf("invalid client status for openIracingFile status %d", ir.status)
	}

	// An iracing file only exists if iRacing is actually running
	ir.logger.Debug("opening iracing memory mapped file", zap.String("filename", iracingMemoryMappedFileName))
	ptrName, err := windows.UTF16PtrFromString(iracingMemoryMappedFileName)
	if err != nil {
		ir.logger.Error("Error creating windows pointer from file name",
			zap.String("filename", iracingMemoryMappedFileName),
			zap.Error(err))
		return 0, nil
	}
	uPtrName := unsafe.Pointer(ptrName)
	ir.logger.Debug("calling windows function OpenFileMappingW")
	for !ir.stop && ir.status == closed {
		// open the file and get a ptr
		modkernel32 := windows.NewLazyDLL("kernel32.dll")
		procOpenFileMapping := modkernel32.NewProc("OpenFileMappingW")
		winHandle, _, err := procOpenFileMapping.Call(uintptr(4), uintptr(0), uintptr(uPtrName))
		if winHandle == 0 && err != nil {
			ir.logger.Debug("Error opening windows memory mapped file",
				zap.String("filename", iracingMemoryMappedFileName),
				zap.Error(err))
			time.Sleep(time.Duration(ir.retryInterval) * time.Second)
		}
		if winHandle > 0 {
			ir.logger.Debug("got ptr to iracing mem mapped file", zap.Uintptr("ptr", winHandle))
			ir.status = open
			return winHandle, nil
		}
	}
	ir.use(uPtrName) // see use, if wierd stuff happens may have to look at winHandle too
	return 0, nil
}

func (ir *Client) readHeader() error {
	if ir.status != open {
		return fmt.Errorf("invalid client status for readHeader status %d", ir.status)
	}
	// parse headerSlice into a new IRHeader struct
	ir.header = newHeader(ir.irMemMapedFile)

	if ir.sessionInfoTickCount != ir.header.SessionInfoTickCount {
		ir.logger.Debug("Session info is new. Read session info.", zap.Int("oldTickCount", ir.sessionInfoTickCount), zap.Int("newTickCount", ir.header.SessionInfoTickCount))
		ir.logger.Debug("iracing header",
			zap.Int("version", ir.header.Ver),
			zap.Int("status", ir.header.Status),
			zap.Int("tickrate", ir.header.TickRate),
			zap.Int("sessionInfoTickCount", ir.header.SessionInfoTickCount),
			zap.Int("infolength", ir.header.SessionInfoLen),
			zap.Int("infoOffset", ir.header.SessionInfoOffset),
			zap.Int("numVars", ir.header.NumVars),
			zap.Int("varOffset", ir.header.VarHeaderOffset),
			zap.Int("numBuf", ir.header.NumBuf),
			zap.Int("BufLen", ir.header.BufLen))
		ir.sessionInfoTickCount = ir.header.SessionInfoTickCount
		ir.readSession()
	}
	for _, bufInfo := range ir.header.BufInfos {
		ir.logger.Debug("bufInfo", zap.Int("tick", bufInfo.TickCount), zap.Int("offset", bufInfo.BufOffset))
	}
	ir.status = loadedHeader
	return nil
}

func (ir *Client) readVarHeaders() error {
	if ir.status < loadedHeader {
		return fmt.Errorf("invalid client status for readVarHeaders status %d", ir.status)
	}
	// setup a slice around the variable headers data in the iracing ptr based on offet and variable header length
	varHeaderSlice := Mmap{}
	h := varHeaderSlice.Header()
	h.Data = ir.irMemMapedFile + uintptr(ir.header.VarHeaderOffset)
	h.Cap = varHeaderLenth * int(ir.header.NumVars)
	h.Len = varHeaderLenth * int(ir.header.NumVars)

	// initialize a map to store telementry variable headers mapped by name
	varHeaders := make(map[string]*varHeader)

	// loop through the variable headers byte slice to read ir.header.NumVars headers and create a variable header to add to map
	for i := 0; i < int(ir.header.NumVars); i++ {
		b := varHeaderSlice[i*varHeaderLenth : (i+1)*varHeaderLenth]
		h := newVarHeader(b)
		varHeaders[h.name] = h
		// ir.logger.Debug("variable header",
		// 	zap.String("name", h.name),
		// 	zap.String("description", h.desc),
		// 	zap.String("type", h.t.String()),
		// )
	}
	ir.logger.Debug("parsed variable headers", zap.Int("numvars", int(ir.header.NumVars)))
	ir.varHeaders = varHeaders
	ir.status = loadedVarHeaders
	return nil
}

// readVarBuf reads the buf info from the header and determines which
// buf is the active buffer based on the tick count in the header
// once found the current buffer is copied into the client varBuf slice
// varBuf will then be used to access the variables
func (ir *Client) readVarBuf() error {
	if ir.status < loadedVarHeaders {
		return fmt.Errorf("invalid client status for readVarBuf status %d", ir.status)
	}

	// lock the varBuf to prevent multiple go routines accessing while updating
	ir.varBufLock.Lock()
	defer ir.varBufLock.Unlock()

	// selecting buffer
	curBuf := 0
	for i, bufInfo := range ir.header.BufInfos {
		ir.logger.Debug("bufInfo", zap.Int("tickCount", bufInfo.TickCount), zap.Int("lastTick", ir.varBufTickCount))
		// is this buffer from a more recent tick count
		if bufInfo.TickCount > ir.varBufTickCount {
			curBuf = i
			ir.varBufTickCount = bufInfo.TickCount
		}
	}
	ir.logger.Debug("reading variable buffer", zap.Int("currentBuffer", curBuf))

	varBuffer := Mmap{}
	h := varBuffer.Header()
	h.Data = ir.irMemMapedFile + uintptr(ir.header.BufInfos[curBuf].BufOffset)
	h.Cap = ir.header.BufLen
	h.Len = ir.header.BufLen
	ir.varBuf = varBuffer
	ir.status = loadedVarBuf
	return nil
}

func (ir *Client) readFloat32Var(varName string) float32 {
	if ir.status != loadedVarBuf {
		return 0
	}

	// before reading vars make sure buffer isn't rewritten
	ir.varBufLock.Lock()
	defer ir.varBufLock.Unlock()

	vH := ir.varHeaders[varName]
	if vH == nil {
		return 0
	}

	raw := binary.LittleEndian.Uint32(ir.varBuf[vH.offset : vH.offset+4])
	return math.Float32frombits(raw)
}

func (ir *Client) readSession() error {
	if ir.status > loadedHeader {
		return fmt.Errorf("invalid client status for readSession status %d", ir.status)
	}
	// setup a slice around the area of the file with the session data in it
	// we'll find a more efficient way to deal with this once its working
	sessionInfoSlice := Mmap{}
	// set the slice header to make this slice point to our data
	h := sessionInfoSlice.Header()
	h.Data = ir.irMemMapedFile + uintptr(ir.header.SessionInfoOffset)
	h.Cap = int(ir.header.SessionInfoLen)
	h.Len = int(ir.header.SessionInfoLen)

	// h.Cap = int(ir.irHeader.SessionInfoLen + ir.irHeader.SessionInfoOffset)
	// h.Len = int(ir.irHeader.SessionInfoLen + ir.irHeader.SessionInfoOffset)

	// TODO Working here trying to read the session info data. Having issues with the size of the info data
	infoStr := windows.ByteSliceToString(sessionInfoSlice)

	ir.SessionInfoYaml = infoStr
	return nil
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
