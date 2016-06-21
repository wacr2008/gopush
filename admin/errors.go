package main

import (
	"errors"
)

var (
	
	ErrSubREQ    = errors.New("sub rpc request error")
	ErrUnSubREQ = errors.New("unsub rpc request error")
	
	ErrUserExist = errors.New("user already exist")
	ErrUserNotExist = errors.New("user does not exist")
	ErrServerNotExist = errors.New("server does not exist")

	ErrUnknowStorage = errors.New("unknow storage")
	ErrStorageInitError = errors.New("init storage failed")
	ErrStorageNodeNotExist = errors.New("storage node does not exist")
)
