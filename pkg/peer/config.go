package peer

import "time"

type P2PConfig struct {
	AdvertiseNameSpace string
	BootstrapPeers     []string
	ProtocolID         string
	ListenPort         int
	MaxConnectedPeers  int

	KeepAliveDuration time.Duration
}

type MessageConfig struct {
	MessageArrivedOb MessageArrivedObserver
	MessageHelper    MessageHelper
}

type Config struct {
	P2PConfig
	MessageConfig
}
