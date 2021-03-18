package talk

import (
	"context"
	"errors"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/sgostarter/libp2p/pkg/p2pio"
)

func Talk(ctx context.Context, h interface{}, peerID, protocolID string,
	StreamTalk func(rw *p2pio.ReadWriteCloser, chExit chan interface{})) error {
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

	chExit := make(chan interface{})
	StreamTalk(p2pio.NewReadWriteCloser(stream), chExit)
	<-chExit
	return nil
}
