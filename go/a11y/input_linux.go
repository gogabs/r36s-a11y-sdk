//go:build linux

package a11y

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

// rawEvent é um struct input_event já decodificado.
type rawEvent struct {
	evType, code uint16
	value        int32
	ms           int64
}

const inputEventSize = 24 // timeval (2 x int64) + type + code + value, em arm64/amd64

func eviocgabs(axis uint16) uintptr {
	// _IOR('E', 0x40 + axis, struct input_absinfo) — input_absinfo tem 6 x int32
	return uintptr(2<<30 | 24<<16 | 'E'<<8 | (0x40 + uint32(axis)))
}

// openGamepads abre todos os /dev/input/event* sem exclusividade (sem
// EVIOCGRAB: o a11yd precisa ver os mesmos botões) e envia os eventos no canal.
func openGamepads(state *inputState, start time.Time, out chan<- rawEvent) int {
	paths, _ := filepath.Glob("/dev/input/event*")
	sort.Strings(paths)
	opened := 0
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		opened++
		for code := range axisCodes {
			var info [6]int32
			_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), eviocgabs(code),
				uintptr(unsafe.Pointer(&info[0])))
			if errno == 0 && info[2] > info[1] {
				state.setAxisRange(code, info[1], info[2])
			}
		}
		go func(f *os.File) {
			defer f.Close()
			buf := make([]byte, inputEventSize*64)
			for {
				n, err := f.Read(buf)
				if err != nil {
					return
				}
				for i := 0; i+inputEventSize <= n; i += inputEventSize {
					b := buf[i : i+inputEventSize]
					sec := int64(binary.LittleEndian.Uint64(b[0:]))
					usec := int64(binary.LittleEndian.Uint64(b[8:]))
					ev := rawEvent{
						evType: binary.LittleEndian.Uint16(b[16:]),
						code:   binary.LittleEndian.Uint16(b[18:]),
						value:  int32(binary.LittleEndian.Uint32(b[20:])),
						ms:     (sec*1000 + usec/1000) - start.UnixMilli(),
					}
					out <- ev
				}
			}
		}(f)
	}
	return opened
}
