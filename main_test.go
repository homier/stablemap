package stablemap

import "unsafe"

//go:nocheckptr
func unsafeConvertSlice[Dest any, Src any](s []Src) []Dest {
	return unsafe.Slice((*Dest)(unsafe.Pointer(unsafe.SliceData(s))), len(s))
}
