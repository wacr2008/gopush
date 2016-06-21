package main

import (
	"sync"
	"gopush/libs/proto"
)


// Bucket is a holder of channels 
type Bucket struct {
	sync.RWMutex			
	channels	map[int64]*Channel		// userid -> channel
	Config		ConfigBucket
}


func NewBucket(config ConfigBucket) (b *Bucket) {
	b = new(Bucket)
	b.channels = make(map[int64]*Channel, config.Channel.Size)
	b.Config = config

	return
}

func (b *Bucket) Put(userid int64, ch *Channel) error {

	b.Lock()
	b.channels[userid] = ch
	b.Unlock()

	return nil
}

func (b *Bucket) DelSafe(userid int64, och *Channel)  {
	var (
		ch *Channel
		ok bool
	)
	b.Lock()
	if ch, ok = b.channels[userid]; ok && och == ch {
		delete(b.channels, userid)
	}
	b.Unlock()
	 
}

func (b *Bucket) Channel(userid int64) *Channel {
	b.RLock()
	ch, ok := b.channels[userid]
	b.RUnlock()
	if !ok {
		return nil
	}
	return ch
}

func (b *Bucket) Push(userid int64, p *proto.Proto) (err error) {
	channel := b.Channel(userid)
	if channel != nil {
		err = channel.Push(p)
	}
	return
}



// ### broadcast ignore error
func (b *Bucket) Broadcast(p *proto.Proto) {
	var ch *Channel
	b.RLock()
	for _, ch = range b.channels {
		ch.Push(p)
	}
	b.RUnlock()
}

func (b *Bucket) BroadcastTopic(topic int32, p *proto.Proto) {
	var ch *Channel
	b.RLock()
	for _, ch = range b.channels {
		if ch.ContainTopic(topic) {
			ch.Push(p)
		}
	}
	b.RUnlock()
}

