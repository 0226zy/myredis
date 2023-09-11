package core

type SdsHdr struct {
	Len  int64
	Free int64
	Buf  []byte
}
