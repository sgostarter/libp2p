package peer

import "io"

type Message interface {
	Bytes() []byte
}

type MessageArrivedObserver interface {
	OnDataArrived(string, Message)
}

type MessageHelper interface {
	ReadMessage(reader io.Reader) (Message, error)
	CreatePingMessage(peerID string) (Message, error)
	CreatePongMessage(pingMessage Message) (Message, error)
	IsPingMessage(message Message) bool
	IsPongMessage(message Message) bool
}
