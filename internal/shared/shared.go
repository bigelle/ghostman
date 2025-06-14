package shared

import (
	"bytes"
	"strings"
	"sync"
)

var bytesBufPool = &sync.Pool{
	New: func() any {
		return bytes.NewBuffer([]byte{})
	},
}

func BytesBuf() *bytes.Buffer {
	return bytesBufPool.Get().(*bytes.Buffer)
}

func PutBytesBuf(b *bytes.Buffer) {
	b.Reset()
	bytesBufPool.Put(b)
}

var stringBuilderPool = &sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

func StringBuilder() *strings.Builder {
	return stringBuilderPool.Get().(*strings.Builder)
}

func PutStringBuilder(b *strings.Builder) {
	b.Reset()
	stringBuilderPool.Put(b)
}

var bytesPool = &sync.Pool{
	New: func() any {
		b := make([]byte, 4 * 1024)
		return &b
	},
}

var bytesMaxCap = 64 * 1024

func Bytes() *[]byte {
	return bytesPool.Get().(*[]byte)
}

func PutBytes(b *[]byte) {
	*b = (*b)[:0]
	if cap(*b) > bytesMaxCap {
		return
	}
	bytesPool.Put(b)
}
