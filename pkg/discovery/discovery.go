package discovery

import (
	"bufio"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"github.com/sgostarter/libp2p/pkg/bootstrap"
)

type Observer interface {
	NewHost(h interface{}, hID string)
	StreamTalk(rw *bufio.ReadWriter)
	OnNewPeerStart()
	OnNewPeer(peerID string)
	OnNewPeerFinish()
}

type ServerParam struct {
	bootstrap.HostParam
	ProtocolID       string
	BootstrapPeers   []string
	AdvertiseNS      string
	MinCheckInterval time.Duration
}

func doBootstrap(ctx context.Context, h host.Host, bootstrapPeers []string,
	advertiseNS string) (*discovery.RoutingDiscovery, error) {
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		return nil, err
	}

	err = kademliaDHT.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	bootstrapAddress := make([]multiaddr.Multiaddr, 0)
	if bootstrapPeers == nil {
		bootstrapAddress = dht.DefaultBootstrapPeers
	} else {
		for _, bootstrapPeer := range bootstrapPeers {
			ma, err := multiaddr.NewMultiaddr(bootstrapPeer)
			if err != nil {
				loge.Errorf(nil, "addr convert failed: %v", err)
				return nil, err
			}
			bootstrapAddress = append(bootstrapAddress, ma)
		}
	}

	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapAddress {
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			loge.Errorf(ctx, "addr to info failed: %v, %v", err, peerAddr)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerInfo); err != nil {
				loge.Warnf(ctx, "connect %v failed: %v", peerInfo.ID.Pretty(), err)
			} else {
				loge.Infof(ctx, "Connection established with bootstrap node: %v", peerInfo)
			}
		}()
	}
	wg.Wait()

	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, advertiseNS)

	return routingDiscovery, nil
}

func RunServer(ctx context.Context, param ServerParam, ob Observer) error {
	if ob == nil {
		return errors.New("no observer")
	}

	h, err := bootstrap.NewHost(ctx, &param.HostParam, libp2p.EnableRelay(relay.OptHop))
	if err != nil {
		return nil
	}
	defer func() {
		_ = h.Close()
	}()
	ob.NewHost(h, h.ID().Pretty())

	h.SetStreamHandler(protocol.ID(param.ProtocolID), func(stream network.Stream) {
		defer func() {
			_ = stream.Close()
		}()
		ob.StreamTalk(bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream)))
	})

	routingDiscovery, err := doBootstrap(ctx, h, param.BootstrapPeers, param.AdvertiseNS)
	if err != nil {
		return err
	}

	timeNow := time.Now()
	minCheckInterval := param.MinCheckInterval
	if minCheckInterval <= 0 {
		minCheckInterval = 5 * time.Second
	}
	var quit bool
	for !quit {
		if time.Since(timeNow) <= minCheckInterval {
			time.Sleep(minCheckInterval)
			timeNow = time.Now()
		}
		peerChan, err := routingDiscovery.FindPeers(ctx, param.AdvertiseNS)
		if err != nil {
			loge.Errorf(ctx, "find peers failed: %v", err)
			continue
		}

		ob.OnNewPeerStart()
		for p := range peerChan {
			if p.ID == h.ID() {
				continue
			}
			ob.OnNewPeer(p.ID.Pretty())
		}
		ob.OnNewPeerFinish()

		select {
		case <-ctx.Done():
			quit = true
		default:
		}
	}
	return nil
}
