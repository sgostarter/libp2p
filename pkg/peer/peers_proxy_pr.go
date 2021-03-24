package peer

import (
	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/jiuzhou-zhao/go-fundamental/structs/tools"
)

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

	ec tools.ExistsChecker
}

func newPR() *PR {
	return &PR{
		peers:           make(map[string]PeerProxy),
		chAddPeer:       make(chan PeerProxy, 2),
		chDelPeer:       make(chan PeerProxy, 2),
		chUpdateIdleIDs: make(chan []string),
		chDoRequest:     make(chan *prRequest, 2),
		chDoAny:         make(chan func(), 2),
		ec:              tools.NewExistsCheckerWithMaxSize(99999999),
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
			loge.Debug(nil, "peersRoutine add peer begin")
			impl.prAddPeer(peer)
			loge.Debug(nil, "peersRoutine add peer end")
		case peer := <-impl.pr.chDelPeer:
			loge.Debug(nil, "peersRoutine del peer begin")
			impl.prDelPeer(peer)
			loge.Debug(nil, "peersRoutine del peer end")
		case req := <-impl.pr.chDoRequest:
			loge.Debug(nil, "peersRoutine do request begin")
			impl.prDoRequest(req)
			loge.Debug(nil, "peersRoutine do request end")
		case fn := <-impl.pr.chDoAny:
			loge.Debug(nil, "peersRoutine do any begin")
			fn()
			loge.Debug(nil, "peersRoutine do any end")
		case idleIDs := <-impl.pr.chUpdateIdleIDs:
			loge.Debug(nil, "peersRoutine update idle peer ids begin")
			impl.prUpdateIdlePeerIDs(idleIDs)
			loge.Debug(nil, "peersRoutine update idle peer ids end")
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
			if req.msg.GossipFlag() {
				impl.pr.ec.Add(req.msg.ID())
			}
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
