package iracing

import (
	"reflect"
	"unsafe"
)

type Mmap []byte

func (m *Mmap) Header() *reflect.SliceHeader {
	return (*reflect.SliceHeader)(unsafe.Pointer(m))
}
