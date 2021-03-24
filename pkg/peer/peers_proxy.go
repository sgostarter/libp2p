package peer

import (
	"context"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sgostarter/libp2p/pkg/bootstrap"
	"github.com/sgostarter/libp2p/pkg/discovery"
	"github.com/sgostarter/libp2p/pkg/p2pio"
)

type PeersProxy interface {
	DoRequest(peerID string, req Message)
	ListPeers(func(peerIDs []string))
	GetID() string
	Wait4Ready()
}

type peerInfo struct {
	peer   PeerProxy
	chExit chan interface{}
}

func NewPeersProxy(ctx context.Context, cfg *Config) PeersProxy {
	if cfg.MessageArrivedOb == nil || cfg.MessageHelper == nil {
		return nil
	}
	peersProxy := &peersProxyImpl{
		ctx:              ctx,
		cfg:              cfg,
		messageArrivedOb: cfg.MessageArrivedOb,
		messageHelper:    cfg.MessageHelper,
		pmr:              newPMR(),
		pr:               newPR(),
		chInitComplete:   make(chan error, 10),
	}

	go peersProxy.p2pDiscoveryRoutine()
	go peersProxy.peersManagerRoutine()
	go peersProxy.peersRoutine()

	return peersProxy
}

type peersProxyImpl struct {
	ctx              context.Context
	cfg              *Config
	messageArrivedOb MessageArrivedObserver
	messageHelper    MessageHelper

	// pmr
	pmr *PMR
	// pr
	pr *PR

	// p2p
	host           interface{}
	hostID         string
	chInitComplete chan error

	cachedPeerIDs []string
}

func (impl *peersProxyImpl) p2pDiscoveryRoutine() {
	loge.Info(impl.ctx, "p2p discovery routine enter")
	err := discovery.RunServer(impl.ctx, discovery.ServerParam{
		HostParam: bootstrap.HostParam{
			ListenPort: impl.cfg.ListenPort,
		},
		ProtocolID:       impl.cfg.ProtocolID,
		BootstrapPeers:   impl.cfg.BootstrapPeers,
		AdvertiseNS:      impl.cfg.AdvertiseNameSpace,
		MinCheckInterval: 0,
	}, impl)
	if err != nil {
		loge.Warnf(impl.ctx, "p2p discovery routine exit with error: %v", err)
	}
	loge.Info(impl.ctx, "p2p discovery routine leave")
}

func (impl *peersProxyImpl) PeerClosed(peer PeerProxy) {
	impl.pmr.chPeerClosed <- peer
}

func (impl *peersProxyImpl) OnDataArrived(peer PeerProxy, req Message) {
	if req.GossipFlag() {
		if !impl.pr.ec.Exists(req.ID()) {
			impl.DoRequest("", req)
			loge.Info(nil, "gossip request to other")
		} else {
			loge.Info(nil, "gossip already request")
		}
	}
	impl.messageArrivedOb.OnDataArrived(peer.GetPeerID(), req)
}

func (impl *peersProxyImpl) DoRequest(peerID string, req Message) {
	impl.pr.chDoRequest <- &prRequest{
		peerID: peerID,
		msg:    req,
	}
}

func (impl *peersProxyImpl) ListPeers(fn func(peerIDs []string)) {
	impl.pr.chDoAny <- func() {
		peerIDs := make([]string, 0, len(impl.pr.peers)+len(impl.pr.idlePeerIDs))
		for peerID := range impl.pr.peers {
			peerIDs = append(peerIDs, peerID)
		}
		peerIDs = append(peerIDs, impl.pr.idlePeerIDs...)
		fn(peerIDs)
	}
}

func (impl *peersProxyImpl) GetID() string {
	return impl.hostID
}

func (impl *peersProxyImpl) Wait4Ready() {
	if impl.hostID != "" {
		return
	}
	<-impl.chInitComplete
}

//
// discovery.Observer
//
func (impl *peersProxyImpl) NewHost(h interface{}, hID string) {
	impl.host = h
	impl.hostID = hID
	impl.chInitComplete <- nil
}

func (impl *peersProxyImpl) StreamTalk(peerID string, rw *p2pio.ReadWriteCloser, chExit chan interface{}) {
	impl.pmr.chNewActivePeer <- &pmrNewActivePeer{
		peerID: peerID,
		rw:     rw,
		chExit: chExit,
	}
}

func (impl *peersProxyImpl) OnNewPeerStart() {
	impl.cachedPeerIDs = nil
}

func (impl *peersProxyImpl) OnNewPeer(peerID string) {
	impl.cachedPeerIDs = append(impl.cachedPeerIDs, peerID)
}

func (impl *peersProxyImpl) OnNewPeerFinish() {
	impl.pmr.chPeersListUpdate <- impl.cachedPeerIDs
	impl.cachedPeerIDs = nil
}
