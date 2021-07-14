package iracing

import (
	"fmt"
	"testing"

	"golang.org/x/sys/windows"
)

func TestNulTerminatedString(t *testing.T) {
	b := []byte{97, 98, 99, 0} // abc nul terminated string
	s := windows.ByteSliceToString(b)

	if len(s) != 3 {
		t.Fail()
	}
	fmt.Println(string(b))

}
