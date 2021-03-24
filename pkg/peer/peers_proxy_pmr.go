package peer

import (
	"errors"
	"time"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sgostarter/libp2p/pkg/p2pio"
	"github.com/sgostarter/libp2p/pkg/talk"
)

type pmrNewActivePeer struct {
	peerID string
	rw     *p2pio.ReadWriteCloser
	chExit chan interface{}
}

type PMR struct {
	peers             map[string]*peerInfo
	peerIdleIDs       map[string]interface{}
	chPeerClosed      chan PeerProxy
	chPeersListUpdate chan []string
	chNewActivePeer   chan *pmrNewActivePeer
	chDoSlowRequest   chan *prRequest
}

func newPMR() *PMR {
	return &PMR{
		peers:             make(map[string]*peerInfo),
		peerIdleIDs:       make(map[string]interface{}),
		chPeerClosed:      make(chan PeerProxy, 2),
		chPeersListUpdate: make(chan []string, 2),
		chNewActivePeer:   make(chan *pmrNewActivePeer),
		chDoSlowRequest:   make(chan *prRequest),
	}
}

func (impl *peersProxyImpl) peersManagerRoutine() {
	loge.Info(impl.ctx, "peers manager routine enter")

	idleTimeout := 2 * time.Minute
	idleTicker := time.NewTicker(idleTimeout)

	loop := true
	for loop {
		select {
		case <-impl.ctx.Done():
			loop = false
		case <-idleTicker.C:
			loge.Debug(nil, "peersManagerRoutine regular peers begin")
			impl.pmrRegularPeers()
			loge.Debug(nil, "peersManagerRoutine regular peers end")
		case peerIDs := <-impl.pmr.chPeersListUpdate:
			if len(impl.pmr.chPeersListUpdate) > 0 {
				continue
			}
			loge.Debug(nil, "peersManagerRoutine list update begin")
			newPeerIDs := make(map[string]interface{})
			for _, peerID := range peerIDs {
				newPeerIDs[peerID] = true
			}
			// remove the invalid peers
			for peerID := range impl.pmr.peers {
				if _, ok := newPeerIDs[peerID]; !ok {
					impl.pmrRemovePeer(peerID)
				} else {
					delete(newPeerIDs, peerID)
				}
			}
			impl.pmr.peerIdleIDs = newPeerIDs
			impl.pmrUpdateIdlePeerIDs()
			impl.pmrRegularPeers()

			idleTicker.Reset(idleTimeout)

			loge.Debug(nil, "peersManagerRoutine list update end")
		case peer := <-impl.pmr.chPeerClosed:
			loge.Debug(nil, "peersManagerRoutine peer close begin")
			if oPeer, ok := impl.pmr.peers[peer.GetPeerID()]; ok {
				if oPeer.peer != peer {
					continue
				}
			}
			impl.pmrRemovePeer(peer.GetPeerID())
			loge.Debug(nil, "peersManagerRoutine peer close end")
		case aPeer := <-impl.pmr.chNewActivePeer:
			loge.Debug(nil, "peersManagerRoutine new active peer begin")
			impl.pmrAddPeer(aPeer.peerID, aPeer.chExit, aPeer.rw)
			loge.Debug(nil, "peersManagerRoutine new active peer end")
		case req := <-impl.pmr.chDoSlowRequest:
			loge.Debug(nil, "peersManagerRoutine do slow request begin")
			impl.pmrDoRequest(req)
			loge.Debug(nil, "peersManagerRoutine do slow request end")
		}
	}

	loge.Info(impl.ctx, "peers manager routine leave")
}

func (impl *peersProxyImpl) pmrUpdateIdlePeerIDs() {
	ids := make([]string, 0, len(impl.pmr.peerIdleIDs))
	for peerID := range impl.pmr.peerIdleIDs {
		ids = append(ids, peerID)
	}
	impl.pr.chUpdateIdleIDs <- ids
}

func (impl *peersProxyImpl) pmrDoRequest(req *prRequest) {
	req.executeCnt++
	if _, ok := impl.pr.peers[req.peerID]; ok {
		impl.pr.chDoRequest <- req
	} else {
		_, err := impl.pmrConnect(req.peerID)
		if err == nil {
			impl.pr.chDoRequest <- req
		}
	}
}

func (impl *peersProxyImpl) pmrRegularPeers() {
	if len(impl.pmr.peerIdleIDs) == 0 {
		return
	}

	cnt := len(impl.pmr.peerIdleIDs)
	if impl.cfg.P2PConfig.MaxConnectedPeers > 0 {
		if len(impl.pmr.peers) >= impl.cfg.P2PConfig.MaxConnectedPeers {
			return
		}
		cnt = impl.cfg.P2PConfig.MaxConnectedPeers - len(impl.pmr.peers)
	}
	for peerID := range impl.pmr.peerIdleIDs {
		_, err := impl.pmrConnect(peerID)
		if err == nil {
			cnt--
			delete(impl.pmr.peerIdleIDs, peerID)
		}
		if cnt <= 0 {
			break
		}
	}

	impl.pmrUpdateIdlePeerIDs()
}

// nolint: unparam
func (impl *peersProxyImpl) pmrConnect(peerID string) (PeerProxy, error) {
	if peerID == "" {
		return nil, errors.New("no peer id")
	}
	if peerInfo, ok := impl.pmr.peers[peerID]; ok {
		return peerInfo.peer, nil
	}
	err := talk.Start(impl.ctx, impl.host, peerID, impl.cfg.ProtocolID, func(peerID string, rw *p2pio.ReadWriteCloser, chExit chan interface{}) {
		impl.pmrAddPeer(peerID, chExit, rw)
	})
	if err != nil {
		return nil, err
	}
	if peerInfo, ok := impl.pmr.peers[peerID]; ok {
		return peerInfo.peer, nil
	}
	return nil, errors.New("no peer")
}

func (impl *peersProxyImpl) pmrAddPeer(peerID string, chExit chan interface{}, rwc *p2pio.ReadWriteCloser) {
	impl.pmrRemovePeer(peerID)

	keepAlive := impl.cfg.KeepAliveDuration
	if keepAlive <= 0 {
		keepAlive = 10 * time.Minute
	}
	impl.pmr.peers[peerID] = &peerInfo{
		peer:   newPeerProxy(impl.ctx, peerID, rwc, impl, impl, impl.messageHelper, keepAlive),
		chExit: chExit,
	}
	impl.pr.chAddPeer <- impl.pmr.peers[peerID].peer
}

func (impl *peersProxyImpl) pmrRemovePeer(peerID string) {
	peerInfo, ok := impl.pmr.peers[peerID]
	if !ok {
		return
	}

	peerInfo.peer.Disconnect()
	delete(impl.pmr.peers, peerID)
	peerInfo.chExit <- true

	impl.pr.chDelPeer <- peerInfo.peer
}
