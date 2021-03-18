package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sgostarter/liblog"
	"github.com/sgostarter/libp2p/pkg/bootstrap"
	"github.com/sgostarter/libp2p/pkg/discovery"
	"github.com/sgostarter/libp2p/pkg/p2pio"
	"github.com/sgostarter/libp2p/pkg/talk"
)

type discoveryObserver struct {
	peers       []string
	peersNewest []string
	h           interface{}
	hID         string
}

func (ob *discoveryObserver) NewHost(h interface{}, hID string) {
	ob.h = h
	ob.hID = hID
	fmt.Println("my host id is ", hID)
}

func (ob *discoveryObserver) StreamTalk(peerID string, rw *p2pio.ReadWriteCloser, chExit chan interface{}) {
	defer func() {
		chExit <- true
	}()
	req, err := rw.ReadString('\n')
	if err != nil {
		loge.Errorf(nil, "read req failed: %v", err)
		return
	}
	resp := "hi, i'm " + ob.hID + " {" + req[:len(req)-1] + "} \n"
	_, err = rw.WriteString(resp)
	if err != nil {
		loge.Errorf(nil, "write resp failed: %v", err)
		return
	}
	_ = rw.Flush()
}

func (ob *discoveryObserver) OnNewPeerStart() {
	ob.peersNewest = nil
}

func (ob *discoveryObserver) OnNewPeer(peerID string) {
	ob.peersNewest = append(ob.peersNewest, peerID)
}

func (ob *discoveryObserver) OnNewPeerFinish() {
	ob.peers = ob.peersNewest
	ob.peersNewest = nil
}

func (ob *discoveryObserver) ListPeers() string {
	ss := &strings.Builder{}
	for _, peer := range ob.peers {
		ss.WriteString(fmt.Sprintf("%s\n", peer))
	}
	ss.WriteString("\n")
	return ss.String()
}

func (ob *discoveryObserver) HostID() string {
	return ob.hID
}

func (ob *discoveryObserver) CheckPeer(peerID, protocolID string) {
	err := talk.Talk(context.Background(), ob.h, peerID, protocolID, func(peerID string, rw *p2pio.ReadWriteCloser, chExit chan interface{}) {
		defer func() {
			chExit <- true
		}()
		_, err := rw.WriteString(fmt.Sprintf("%s-ping\n", ob.hID))
		if err != nil {
			loge.Fatalf(nil, "write string failed: %w", err)
			return
		}
		_ = rw.Flush()

		resp, err := rw.ReadString('\n')
		if err != nil {
			loge.Fatalf(nil, "read string failed: %v", err)
			return
		}
		fmt.Printf("%s\n", resp)
	})
	if err != nil {
		loge.Errorf(nil, "check peer failed: %v", err)
	}
}

func main() {
	var bootstrapMode bool
	flag.BoolVar(&bootstrapMode, "bootstrap", false, "bootstrap mode")

	var ns string
	flag.StringVar(&ns, "ns", "TestAdvertiseNS", "advertise name space")

	var bootstrapPeers string
	flag.StringVar(&bootstrapPeers, "bspeer", "/ip4/172.28.89.156/tcp/4999/p2p/QmfAwEcZ9RpvXhRL1AWg6BSDBuvtzY2qw45KGNx6fZatBR", "boot strap peer")

	var protocolID string
	flag.StringVar(&protocolID, "pid", "testProtocolID/1.0.0", "protocol id")

	var port int
	flag.IntVar(&port, "port", 5000, "port")

	flag.Parse()

	logger, err := liblog.NewZapLogger()
	if err != nil {
		panic(err)
	}
	loge.SetGlobalLogger(loge.NewLogger(logger))

	if bootstrapMode {
		go func() {
			err := bootstrap.RunServer(context.Background(), bootstrap.ServerParam{
				HostParam: bootstrap.HostParam{
					ListenPort:  port - 1,
					UseIdentity: true,
					PriKeyFile:  "priKey.dat",
				},
			})
			if err != nil {
				panic(err)
			}
		}()
	}

	ob := &discoveryObserver{}

	go func() {
		_ = discovery.RunServer(context.Background(), discovery.ServerParam{
			HostParam: bootstrap.HostParam{
				ListenPort:  port,
				UseIdentity: false,
				PriKeyFile:  "",
			},
			ProtocolID:       protocolID,
			BootstrapPeers:   strings.Split(bootstrapPeers, ";"),
			AdvertiseNS:      ns,
			MinCheckInterval: 0,
		}, ob)
	}()

	fnReadString := func(r *bufio.Reader) string {
		cmd, err := r.ReadString('\n')
		if err != nil {
			loge.Errorf(nil, "read console string failed: %v", err)
			return ""
		}

		return strings.Trim(cmd, "\r\n\t ")
	}

	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n> ")
		switch fnReadString(stdReader) {
		case "list":
			fmt.Println(ob.ListPeers())
		case "check":
			fmt.Print("input the peerID:")
			peerID := fnReadString(stdReader)
			fmt.Println("")
			ob.CheckPeer(peerID, protocolID)
		case "print":
			fmt.Printf("my id is %v", ob.HostID())
		default:
		}
	}
}
