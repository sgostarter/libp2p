package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"github.com/sgostarter/libp2p/pkg/identity"
)

type HostParam struct {
	ListenPort  int
	UseIdentity bool
	PriKeyFile  string
}

type ServerParam struct {
	HostParam
}

func getPriKeyFromFile(priKeyFile string) (crypto.PrivKey, error) {
	if priKeyFile == "" {
		return nil, errors.New("no pri key file")
	}
	priKey, _, err := identity.LoadKeyPair(priKeyFile, "")
	if err == nil {
		return priKey, nil
	}

	err = identity.NewKeyPair(priKeyFile, "0")
	if err != nil {
		return nil, err
	}
	priKey, _, err = identity.LoadKeyPair(priKeyFile, "")
	if err != nil {
		return nil, err
	}
	return priKey, nil
}

func NewHost(ctx context.Context, hostParam *HostParam, opts ...libp2p.Option) (host.Host, error) {
	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", hostParam.ListenPort))
	if err != nil {
		return nil, err
	}
	opts = append(opts, libp2p.ListenAddrs(sourceMultiAddr))
	if hostParam.UseIdentity {
		priKey, err := getPriKeyFromFile(hostParam.PriKeyFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, libp2p.Identity(priKey))
	}
	return libp2p.New(ctx, opts...)
}

func RunServer(ctx context.Context, param ServerParam) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	h, err := NewHost(ctx, &param.HostParam)
	if err != nil {
		return fmt.Errorf("new instance failed: %w", err)
	}

	fmt.Println("This node: ", h.ID().Pretty(), " ", h.Addrs())

	_, err = dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		return fmt.Errorf("new dht failed: %w", err)
	}

	<-ctx.Done()
	return nil
}
