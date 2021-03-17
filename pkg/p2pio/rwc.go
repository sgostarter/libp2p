package p2pio

import (
	"bufio"

	"github.com/libp2p/go-libp2p-core/network"
)

type ReadWriteCloser struct {
	*bufio.ReadWriter
	s network.Stream
}

func NewReadWriteCloser(s network.Stream) *ReadWriteCloser {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	return &ReadWriteCloser{
		ReadWriter: rw,
		s:          s,
	}
}

func (rwc *ReadWriteCloser) Close() error {
	return rwc.s.Close()
}
