package shared

import (
	"bytes"
	"strings"
	"sync"
)

var bytesBufPool = &sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 4*1024))
	},
}

func BytesBuf() *bytes.Buffer {
	buf := bytesBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutBytesBuf(b *bytes.Buffer) {
	if b.Cap() > 64*1024 { 
		return
	}
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
