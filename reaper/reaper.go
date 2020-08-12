package reaper

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func Set() error {
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(1), 0, 0, 0)
}

func Unset() error {
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(0), 0, 0, 0)
}

func IsSubreaper() (bool, error) {
	var i uintptr

	if err := unix.Prctl(unix.PR_GET_CHILD_SUBREAPER, uintptr(unsafe.Pointer(&i)), 0, 0, 0); err != nil {
		return false, err
	}

	return (int(i) != 0), nil
}
