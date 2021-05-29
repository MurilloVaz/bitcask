// +build windows

package flock

import (
	"errors"
	"os"
	sys "syscall"
	"time"
	"unsafe"
)

// LockFileEx code derived from golang build filemutex_windows.go @ v1.5.1
var (
	modkernel32      = sys.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")

	// ErrTimeout is returned when we cannot obtain an exclusive lock
	// on the key file.
	ErrTimeout = errors.New("timeout waiting on lock to become available")
)

const (
	// see https://msdn.microsoft.com/en-us/library/windows/desktop/aa365203(v=vs.85).aspx
	flagLockExclusive       = 2
	flagLockFailImmediately = 1

	// see https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx
	errLockViolation sys.Errno = 0x21
)

func lockFileEx(h sys.Handle, flags, reserved, locklow, lockhigh uint32, ol *sys.Overlapped) (err error) {
	r, _, err := procLockFileEx.Call(uintptr(h), uintptr(flags), uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)))
	if r == 0 {
		return err
	}
	return nil
}

func lock(path string, exclusive bool) (*os.File, error) {
	// Create a separate lock file on windows because a process
	// cannot share an exclusive lock on the same file.
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	for {

		var flag uint32 = flagLockFailImmediately
		if !exclusive {
			flag |= flagLockExclusive
		}
		err = lockFileEx(sys.Handle(file.Fd()), flag, 0, 1, 0, &sys.Overlapped{})
		if err == nil {
			return file, nil
		} else if err != errLockViolation {
			return nil, err
		}

		// Wait for a bit and try again.
		time.Sleep(50 * time.Millisecond)
	}
}

func rm_if_match(fh *os.File, path string) error {
	return os.Remove(path)
}

func lock_sys(path string, nonBlocking bool) (*os.File, error) {

	fh, err := lock(path, nonBlocking)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			fh.Close()
		}
	}()

	return fh, err
}

func sameInodes(f *os.File, path string) bool {
	return true
}
