//go:build amd64

package main

type SigInfoSys struct {
	Signo    int32
	Errno    int32
	Code     int32
	_        [4]byte
	CallAddr uintptr
	Syscall  int32
	Arch     uint32
	_        [96]byte
}
