package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sgostarter/liblog"
	"github.com/sgostarter/libp2p/pkg/peer"
)

//
//
//
type baseMessage struct {
	ID   byte
	Text string
}

func (msg *baseMessage) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(msg.ID)
	_ = gob.NewEncoder(&buf).Encode(msg)
	d := make([]byte, 1024)
	copy(d, buf.Bytes())
	return d
}

type pingMessage struct {
	baseMessage
}

type pongMessage struct {
	baseMessage
}

type textMessage struct {
	baseMessage
}

//
//
//
type messageArrivedObserver struct {
}

func (ob *messageArrivedObserver) OnDataArrived(peerID string, msg peer.Message) {
	textMsg := msg.(*textMessage)
	fmt.Printf("%v talk to me: %v\n", peerID, textMsg.Text)
}

type messageHelper struct {
}

func (mh *messageHelper) ReadMessage(reader io.Reader) (msg peer.Message, err error) {
	buf := make([]byte, 1024)
	_, err = reader.Read(buf)
	if err != nil {
		return
	}

	r := bytes.NewReader(buf)
	id, err := r.ReadByte()
	if err != nil {
		return
	}

	if id == 0x01 {
		var ping pingMessage
		err = gob.NewDecoder(r).Decode(&ping)
		if err != nil {
			return
		}
		msg = &ping
		return
	}

	if id == 0x02 {
		var pong pongMessage
		err = gob.NewDecoder(r).Decode(&pong)
		if err != nil {
			return
		}
		msg = &pong
		return
	}

	if id == 0x03 {
		var text textMessage
		err = gob.NewDecoder(r).Decode(&text)
		if err != nil {
			return
		}
		msg = &text
		return
	}

	err = fmt.Errorf("unknown id: %v", id)

	return
}

func (mh *messageHelper) CreatePingMessage(peerID string) (peer.Message, error) {
	return &pingMessage{
		baseMessage: baseMessage{
			ID:   0x01,
			Text: "PING",
		},
	}, nil
}

func (mh *messageHelper) CreatePongMessage(pMsg peer.Message) (peer.Message, error) {
	return &pongMessage{
		baseMessage: baseMessage{
			ID:   0x01,
			Text: "PONG",
		},
	}, nil
}

func (mh *messageHelper) IsPingMessage(pMsg peer.Message) bool {
	_, ok := pMsg.(*pingMessage)
	return ok
}

func (mh *messageHelper) IsPongMessage(pMsg peer.Message) bool {
	_, ok := pMsg.(*pongMessage)
	return ok
}

func main() {
	logger, err := liblog.NewZapLogger()
	if err != nil {
		panic(err)
	}
	loge.SetGlobalLogger(loge.NewLogger(logger))

	cfg := peer.Config{
		P2PConfig: peer.P2PConfig{
			AdvertiseNameSpace: "chat_hoho_992121s1",
			BootstrapPeers:     nil,
			ProtocolID:         "chat.chat.chat",
			ListenPort:         0,
			MaxConnectedPeers:  0,
			KeepAliveDuration:  time.Minute,
		},
		MessageConfig: peer.MessageConfig{
			MessageArrivedOb: &messageArrivedObserver{},
			MessageHelper:    &messageHelper{},
		},
	}
	peersProxy := peer.NewPeersProxy(context.Background(), &cfg)

	fnReadString := func(r *bufio.Reader) string {
		fmt.Print("\n> ")
		cmd, err := r.ReadString('\n')
		if err != nil {
			loge.Errorf(nil, "read console string failed: %v", err)
			return ""
		}

		return strings.Trim(cmd, "\r\n\t ")
	}

	stdReader := bufio.NewReader(os.Stdin)

	for {
		switch v := fnReadString(stdReader); v {
		case "list":
			peersProxy.ListPeers(func(peerIDs []string) {
				fmt.Println("")
				fmt.Println("-- list peers begin")
				for _, peerID := range peerIDs {
					fmt.Println(peerID)
				}
				fmt.Println("-- list peers end")
				fmt.Print("\n> ")
			})
		case "print":
			fmt.Println("my host id is ", peersProxy.GetID())
		default:
			var text, receiver string
			n, err := fmt.Sscanf(v, "send %s to %s", &text, &receiver)
			if err != nil {
				continue
			}
			if n != 2 {
				continue
			}
			if receiver == "all" {
				receiver = ""
			}
			peersProxy.DoRequest(receiver, &textMessage{baseMessage{
				ID:   0x03,
				Text: text,
			}})
		}
	}
}
