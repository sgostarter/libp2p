package main

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/sgostarter/libp2p/pkg/peer"
	"github.com/stretchr/testify/assert"
)

func TestIs(t *testing.T) {
	mh := &messageHelper{}
	m, err := mh.CreatePingMessage("a")
	assert.Nil(t, err)
	ok := mh.IsPingMessage(m)
	assert.True(t, ok)
	ok = mh.IsPongMessage(m)
	assert.False(t, ok)
}

func TestGob(t *testing.T) {
	gob.Register(baseMessage{})
	gob.Register(nickNameSetMessage{})
	nn := &nickNameSetMessage{
		baseMessage: baseMessage{
			MsgID: 4,
			Text:  "aaaaaaaaaaaaaaaaaaaaaaaaaa",
			UUID:  "sssssssssssssss",
		},
		PeerID:   "bbbbbbbbbbb",
		NickName: "ccccccccccccc",
	}

	msg := peer.Message(nn)

	bs := msg.Bytes()

	mh := &messageHelper{}
	nn2, err := mh.ReadMessage(bytes.NewReader(bs))
	assert.Nil(t, err)
	nn3 := nn2.(*nickNameSetMessage)

	assert.True(t, nn3.MsgID == 4)
}
