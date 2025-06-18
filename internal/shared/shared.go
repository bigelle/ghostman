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

