package iracing

import (
	"reflect"
	"unsafe"
)

type Mmap []byte

// Header function exposes the slice header so we can manipulate it around another slice
// that other slice is a portion of the memory mapped iracing file
func (m *Mmap) Header() *reflect.SliceHeader {
	return (*reflect.SliceHeader)(unsafe.Pointer(m))
}
