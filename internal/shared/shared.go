package shared

import (
	"bytes"
	"sync"
)

var BytesBufPool = &sync.Pool{
	New: func() any {
		return bytes.NewBuffer([]byte{})
	},
}
