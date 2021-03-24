package peer

import (
	"context"
	"time"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sgostarter/libp2p/pkg/p2pio"
)

type closeObserver interface {
	PeerClosed(PeerProxy)
}

type messageArrivedObserver interface {
	OnDataArrived(PeerProxy, Message)
}

// nolint: golint
type PeerProxy interface {
	GetPeerID() string
	DoRequest(req Message)
	Disconnect()
}

type peerProxyImpl struct {
	ctx              context.Context
	peerID           string
	rwc              *p2pio.ReadWriteCloser
	closeOb          closeObserver
	messageArrivedOb messageArrivedObserver
	messageHelper    MessageHelper
	lastTouch        time.Time
	ch2Write         chan Message
	keepAlive        time.Duration
}

func newPeerProxy(ctx context.Context, peerID string, rwc *p2pio.ReadWriteCloser, closeOb closeObserver,
	messageArrivedOb messageArrivedObserver, messageHelper MessageHelper, keepAlive time.Duration) PeerProxy {
	impl := &peerProxyImpl{
		ctx:              ctx,
		peerID:           peerID,
		rwc:              rwc,
		closeOb:          closeOb,
		messageArrivedOb: messageArrivedOb,
		messageHelper:    messageHelper,
		lastTouch:        time.Now(),
		ch2Write:         make(chan Message, 2),
		keepAlive:        keepAlive,
	}
	go impl.rwRoutine()

	return impl
}

func (impl *peerProxyImpl) rwRoutine() {
	chMsgIncoming := make(chan Message, 2)
	chReadError := make(chan error, 1)

	go func() {
		defer func() {
			_ = impl.rwc.Close()
		}()
		for {
			msg, err := impl.messageHelper.ReadMessage(impl.rwc)
			if err != nil {
				chReadError <- err
				loge.Errorf(impl.ctx, "peer %v read failed: %v", impl.peerID, err)
				break
			}
			loge.Debugf(nil, "-- receive: %v", msg)
			chMsgIncoming <- msg
		}
	}()

	timeoutChecker := time.NewTicker(impl.keepAlive)
	pingTicker := time.NewTicker(impl.keepAlive / 3)

	fnSendPing := func() {
		pingMsg, err := impl.messageHelper.CreatePingMessage(impl.peerID)
		if err != nil {
			loge.Errorf(impl.ctx, "peer %v create ping message failed: %v", impl.peerID, err)
			return
		}
		impl.ch2Write <- pingMsg
	}

	fnSendPong := func(pingMsg Message) {
		pongMsg, err := impl.messageHelper.CreatePongMessage(pingMsg)
		if err != nil {
			loge.Errorf(impl.ctx, "peer %v create pong message failed: %v", impl.peerID, err)
			return
		}
		impl.ch2Write <- pongMsg
	}

	loop := true
	for loop {
		select {
		case <-chReadError:
			loop = false
		case <-timeoutChecker.C:
			loge.Errorf(impl.ctx, "peer %v timeout exit rw routine", impl.peerID)
			loop = false
		case <-pingTicker.C:
			fnSendPing()
		case msg := <-impl.ch2Write:
			_, err := impl.rwc.Write(msg.Bytes())
			if err != nil {
				loge.Errorf(impl.ctx, "write message failed: %v", err)
				break
			}
			_ = impl.rwc.Flush()
			loge.Debugf(nil, "-- send: %v", msg)
		case msg := <-chMsgIncoming:
			if impl.messageHelper.IsPingMessage(msg) {
				fnSendPong(msg)
			} else if impl.messageHelper.IsPongMessage(msg) {
				impl.lastTouch = time.Now()
				timeoutChecker.Reset(10 * time.Minute)
			} else {
				impl.messageArrivedOb.OnDataArrived(impl, msg)
			}
		}
	}
	_ = impl.rwc.Close()

	timeoutChecker.Stop()
	pingTicker.Stop()

	impl.closeOb.PeerClosed(impl)
}

func (impl *peerProxyImpl) GetPeerID() string {
	return impl.peerID
}

func (impl *peerProxyImpl) DoRequest(req Message) {
	impl.ch2Write <- req
}

func (impl *peerProxyImpl) Disconnect() {
	loge.Infof(impl.ctx, "peer %v disconnect", impl.peerID)
	_ = impl.rwc.Close()
}
