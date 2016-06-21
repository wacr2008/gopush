package main

import (
	"errors"
)

var (
	// comet
	ErrComet = errors.New("comet rpc is not available")

	ErrConnectComet = errors.New("connect comet rpc error")

	ErrInvalidMsg = errors.New("invalid message format!")

	ErrUnknowStorage = errors.New("unknow storage type")
ErrStorageInitError = errors.New("init storage error")
ErrStorageNodeNotExist = errors.New("storage node does not exist")
)
