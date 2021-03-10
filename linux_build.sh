#!/bin/sh

GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o ./bins/p2ptools ./cmd/p2ptools/main.go

