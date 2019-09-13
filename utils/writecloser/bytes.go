package writecloser

import (
	"bytes"
)

type BytesBuffer struct {
	*bytes.Buffer
}

func (b *BytesBuffer) Close() error {
	return nil
}
