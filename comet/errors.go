package main

import (
	"errors"
)

var (
	// server
	ErrHandshake       = errors.New("handshake failed")
	ErrOperation       = errors.New("request operation is invalid")
	ErrServerNotListen = errors.New("server not listen")
	
	// ring
	ErrRingBufferEmpty = errors.New("ring buffer empty")
	ErrRingBufferFull  = errors.New("ring buffer full")
	
	// timer
	ErrTimerFull   = errors.New("timer full")
	ErrTimerEmpty  = errors.New("timer empty")
	ErrTimerNoItem = errors.New("timer item does not exist")
	
	// 
	ErrCometPushChannel  = errors.New("comet push channel error")
	ErrCometRpc = errors.New("comet rpc  error")
	ErrCometRpcMsgREQ   = errors.New("rpc msg arg error")
	ErrCometRpcMsgsREQ  = errors.New("rpc msgs arg error")
	ErrMCometRpcMsgREQ  = errors.New("rpc mmsg arg error")
	ErrMCometRpcMsgsREQ = errors.New("rpc mmsgs arg error")
	
	// rpc
	ErrAdminRpc	=	errors.New("admin rpc is not available")
	ErrPushRpc	=	errors.New("push rpc is not available")
	ErrAuthFailed	= 	errors.New("auth failed!")
	ErrChannelNotExist	=	errors.New("channel does not exist!")
	
	ErrInvalidConfig		=	errors.New("invalid configuration")
	
	ErrInternal	=	errors.New("internal error")
)
