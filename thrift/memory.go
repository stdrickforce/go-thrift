package thrift

import "bytes"

type memory_buffer struct {
	ProtocolReader
	ProtocolWriter
	buf *bytes.Buffer
}

func NewMemoryBuffer(p ProtocolBuilder, bs []byte) *memory_buffer {
	mb := &memory_buffer{
		buf: bytes.NewBuffer(bs),
	}
	mb.ProtocolReader = p.NewProtocolReader(mb.buf)
	mb.ProtocolWriter = p.NewProtocolWriter(mb.buf)
	return mb
}

func (self *memory_buffer) Flush() error {
	return nil
}

func (self *memory_buffer) Close() error {
	return nil
}

func (self *memory_buffer) GetBuffer() *bytes.Buffer {
	return self.buf
}
