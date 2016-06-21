package main

import (
	"gopush/libs/bufio"
	"gopush/libs/proto"
)

// Channel : manage user information
type Channel struct {
	ProtoRing	RingBuffer
	Topics		map[int32]struct{}
	Token		string
	signal		chan *proto.Proto
	Writer		bufio.Writer
	Reader 		bufio.Reader
}

func NewChannel(ringSize, channelBufferSize int) *Channel {
	c := new(Channel)
	c.ProtoRing.Init(ringSize)
	c.signal = make(chan *proto.Proto, channelBufferSize)
	c.Topics = make(map[int32]struct{})
	return c
}


// copy, do not ref
func (c *Channel) SetTopics(topics []int32) {
	for _, t := range topics {
		c.Topics[t] = struct{}{}
	}
}

func (c *Channel) GetTopics() []int32 {
	topics := make([]int32,0)
	for t, _ := range c.Topics {
		topics = append(topics,t )
	}
	return topics
}

func (c *Channel) SetToken(token string) {
	c.Token = token
}

func (c *Channel) ContainTopic(topic int32) bool {
	_, ok := c.Topics[topic]
	return ok
}

// push message to user
func (c *Channel) Push(p *proto.Proto) (error) {
	if p.Ensure {
		return c.PushEnsure(p)
	} else {
		return c.PushFast(p)
	}
}

// if buffer is full, message will be discard
func (c *Channel) PushFast(p *proto.Proto) (err error) {
	select {
		case c.signal <- p:
		default:
			err = ErrCometPushChannel
	}
	return
}

// to ensure the message is delivered
func (c *Channel) PushEnsure(p *proto.Proto) (err error) {
	c.signal <- p
	return nil
}

// get a protocol struct from buffer
func (c *Channel) GetProto() (p *proto.Proto) {
	p = <-c.signal
	return
}


// discard all protocol structs
func (c *Channel) DiscardAllProto() {
	for {
		select {
			case <- c.signal:
			default:
			return
		}
	}
}

// 
func (c *Channel) Ready() {
	c.signal <- proto.ProtoReady
}


func (c *Channel) Close() {
	c.signal <- proto.ProtoFinish
}
