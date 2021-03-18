package peer

import "github.com/jiuzhou-zhao/go-fundamental/loge"

type prRequest struct {
	peerID     string
	msg        Message
	executeCnt int
}

type PR struct {
	peers       map[string]PeerProxy
	idlePeerIDs []string

	chAddPeer       chan PeerProxy
	chDelPeer       chan PeerProxy
	chUpdateIdleIDs chan []string
	chDoRequest     chan *prRequest
	chDoAny         chan func()
}

func newPR() *PR {
	return &PR{
		peers:           make(map[string]PeerProxy),
		chAddPeer:       make(chan PeerProxy, 100),
		chDelPeer:       make(chan PeerProxy, 100),
		chUpdateIdleIDs: make(chan []string),
		chDoRequest:     make(chan *prRequest, 100),
		chDoAny:         make(chan func(), 100),
	}
}

func (impl *peersProxyImpl) peersRoutine() {
	loge.Info(impl.ctx, "peer routine enter")
	loop := true
	for loop {
		select {
		case <-impl.ctx.Done():
			loop = false
		case peer := <-impl.pr.chAddPeer:
			impl.prAddPeer(peer)
		case peer := <-impl.pr.chDelPeer:
			impl.prDelPeer(peer)
		case req := <-impl.pr.chDoRequest:
			impl.prDoRequest(req)
		case fn := <-impl.pr.chDoAny:
			fn()
		case idleIDs := <-impl.pr.chUpdateIdleIDs:
			impl.prUpdateIdlePeerIDs(idleIDs)
		}
	}
	loge.Info(impl.ctx, "peer routine leave")
}

func (impl *peersProxyImpl) prUpdateIdlePeerIDs(ids []string) {
	impl.pr.idlePeerIDs = ids
}

func (impl *peersProxyImpl) prDoRequest(req *prRequest) {
	if req.peerID == "" {
		for _, peer := range impl.pr.peers {
			peer.DoRequest(req.msg)
		}
		return
	}
	if peer, ok := impl.pr.peers[req.peerID]; ok {
		peer.DoRequest(req.msg)
	} else {
		loge.Warnf(impl.ctx, "prDoRequest %v failed: no peer", req.peerID)
		req.executeCnt++
		if req.executeCnt >= 10 {
			loge.Errorf(impl.ctx, "prDoRequest failed for %v", req)
			return
		}
		impl.pmr.chDoSlowRequest <- req
	}
}

func (impl *peersProxyImpl) prAddPeer(peer PeerProxy) {
	if oPeer, ok := impl.pr.peers[peer.GetPeerID()]; ok {
		if oPeer == peer {
			return
		}
		impl.prDelPeer(peer)
	}
	impl.pr.peers[peer.GetPeerID()] = peer
}

func (impl *peersProxyImpl) prDelPeer(peer PeerProxy) {
	delete(impl.pr.peers, peer.GetPeerID())
}
