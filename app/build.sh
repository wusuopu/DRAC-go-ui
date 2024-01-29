#!/bin/sh

go build -ldflags "-linkmode external -extldflags=-static" src/main.go
strip main
