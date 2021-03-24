package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	uuid "github.com/satori/go.uuid"
	"github.com/sgostarter/liblog"
	"github.com/sgostarter/libp2p/pkg/peer"
)

//
//
//
type baseMessage struct {
	MsgID byte
	Text  string
	UUID  string
}

func (msg *baseMessage) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(msg.MsgID)

	md, _ := json.Marshal(msg)
	d := make([]byte, 1024)
	copy(d, buf.Bytes())
	copy(d[1:], md)
	return d
}

func (msg *baseMessage) ID() string {
	if msg.UUID == "" {
		msg.UUID = uuid.NewV4().String()
	}
	return fmt.Sprintf("%d_%s", msg.MsgID, msg.UUID)
}

func (msg *baseMessage) GossipFlag() bool {
	return false
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

type nickNameSetMessage struct {
	baseMessage
	PeerID   string
	NickName string
}

func (msg *nickNameSetMessage) GossipFlag() bool {
	return true
}

func (msg *nickNameSetMessage) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(msg.MsgID)

	md, _ := json.Marshal(msg)
	d := make([]byte, 1024)
	copy(d, buf.Bytes())
	copy(d[1:], md)
	return d
}

//
//
//
type messageArrivedObserver struct {
	nickNames map[string]string
}

func newMessageArrivedObserver() *messageArrivedObserver {
	return &messageArrivedObserver{
		nickNames: make(map[string]string),
	}
}

func (ob *messageArrivedObserver) getNickName(peerID string) string {
	if nickName, ok := ob.nickNames[peerID]; ok {
		return nickName
	}
	return peerID
}

func (ob *messageArrivedObserver) OnDataArrived(peerID string, msg peer.Message) {
	if textMsg, ok := msg.(*textMessage); ok {
		fmt.Printf("%v talk to me: %v\n", ob.getNickName(peerID), textMsg.Text)
	} else if nickNameSetMsg, ok := msg.(*nickNameSetMessage); ok {
		ob.nickNames[nickNameSetMsg.PeerID] = nickNameSetMsg.NickName
	}
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
		err = json.NewDecoder(r).Decode(&ping)
		// err = gob.NewDecoder(r).Decode(&ping)
		if err != nil {
			return
		}
		msg = &ping
		return
	}

	if id == 0x02 {
		var pong pongMessage
		err = json.NewDecoder(r).Decode(&pong)
		// err = gob.NewDecoder(r).Decode(&pong)
		if err != nil {
			return
		}
		msg = &pong
		return
	}

	if id == 0x03 {
		var text textMessage
		err = json.NewDecoder(r).Decode(&text)
		// err = gob.NewDecoder(r).Decode(&text)
		if err != nil {
			return
		}
		msg = &text
		return
	}

	if id == 0x04 {
		var nickName nickNameSetMessage
		err = json.NewDecoder(r).Decode(&nickName)
		// err = gob.NewDecoder(r).Decode(&nickName)
		if err != nil {
			return
		}
		msg = &nickName
		return
	}

	err = fmt.Errorf("unknown id: %v", id)

	return
}

func (mh *messageHelper) CreatePingMessage(peerID string) (peer.Message, error) {
	return &pingMessage{
		baseMessage: baseMessage{
			MsgID: 0x01,
			Text:  "PING",
		},
	}, nil
}

func (mh *messageHelper) CreatePongMessage(pMsg peer.Message) (peer.Message, error) {
	return &pongMessage{
		baseMessage: baseMessage{
			MsgID: 0x02,
			Text:  "PONG",
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
	var port int
	flag.IntVar(&port, "port", 0, "port")
	flag.Parse()

	logger, err := liblog.NewZapLogger()
	if err != nil {
		panic(err)
	}
	loge.SetGlobalLogger(loge.NewLogger(logger))

	ob := newMessageArrivedObserver()
	cfg := peer.Config{
		P2PConfig: peer.P2PConfig{
			AdvertiseNameSpace: "chat_demo_4zjz12",
			BootstrapPeers:     nil,
			ProtocolID:         "chat.chat.chat",
			ListenPort:         port,
			MaxConnectedPeers:  0,
			KeepAliveDuration:  time.Minute,
		},
		MessageConfig: peer.MessageConfig{
			MessageArrivedOb: ob,
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
		case "nickname":
			fmt.Print("enter your nickName:> ")
			nickName := fnReadString(stdReader)
			peersProxy.DoRequest("", &nickNameSetMessage{
				baseMessage: baseMessage{
					MsgID: 0x04,
					Text:  "NickName",
				},
				PeerID:   peersProxy.GetID(),
				NickName: nickName,
			})
			ob.nickNames[peersProxy.GetID()] = nickName
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
				MsgID: 0x03,
				Text:  text,
			}})
		}
	}
}
