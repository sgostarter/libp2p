package connect

import (
	"bufio"
	"context"
	"errors"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

func Connect(ctx context.Context, h interface{}, peerID, protocolID string, StreamTalk func(rw *bufio.ReadWriter)) error {
	ho, ok := h.(host.Host)
	if !ok {
		return errors.New("no host")
	}

	p, err := peer.Decode(peerID)
	if err != nil {
		return err
	}

	stream, err := ho.NewStream(ctx, p, protocol.ID(protocolID))
	if err != nil {
		return err
	}
	defer func() {
		_ = stream.Close()
	}()

	StreamTalk(bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream)))
	return nil
}
