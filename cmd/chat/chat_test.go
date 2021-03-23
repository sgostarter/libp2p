package main

import (
	"testing"

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
