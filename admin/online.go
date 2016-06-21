package main

import (
	log "github.com/ikenchina/golog"
	"time"
)
const (
	getmsgsize = int64(100)
	chansize = 8
	chanmod = 7
)


type NotifyChannel struct {
	nchan chan int64
}



func (n *NotifyChannel) subDo(uid int64) {
		
	startIdx := int64(0)
	for {
		msgs, err := DefaultStorage.GetMsg(uid, startIdx, getmsgsize + startIdx)
		//log.Debugln(len(msgs), startIdx, getmsgsize + startIdx)
		if len(msgs) == 0 || err != nil {
			break	
		}
		startIdx += getmsgsize + 1
		
		nowsecond := time.Now().Unix()
		for _, msg := range msgs {
			if !msg.Ensure {
				DefaultStorage.DelMsg(msg)
			}
			s, err := DefaultServer.GetBucket(uid).GetUserServer(uid)
			if err == nil {
				// delete expired message
				if msg.Expire > 0 {
					if msg.Time + msg.Expire <= nowsecond {
						DefaultStorage.DelMsg(msg)
						continue
					}
				}
				// increase times of retry send
				if msg.RedoMax > 0 {
					if msg.RedoCount > msg.RedoMax {
						DefaultStorage.DelMsg(msg)
						continue
					}
					msg.RedoCount++
					DefaultStorage.SaveMsg(msg)
				}
				
				DefaultMQ.Push(s, msg)
			}
		}
		if int64(len(msgs)) < getmsgsize {
			break
		}
		
	}
}

func (n *NotifyChannel) Do() {
	for {
		uid := <- n.nchan 
		count := DefaultStorage.Size(uid)
		if count > getmsgsize  {
			go n.subDo(uid)
		} else {
			n.subDo(uid)
		}
	}
}
func (n *NotifyChannel) Notify(userid int64) {
	n.nchan <- userid
}


func NewNotifyChannel(size int32) *NotifyChannel {
	n := new(NotifyChannel)
	n.nchan = make(chan int64, size)
	go n.Do()
	return n
}

type OnlineNotify struct {
	chans []*NotifyChannel
}

func NewOnlineNotify() *OnlineNotify {
	on := new(OnlineNotify)
	for i:=0; i < chansize; i++ {
		c := NewNotifyChannel(100)
		on.chans = append(on.chans, c)
	}
	log.Infof("init online notify channel size : %d", len(on.chans))
	return on
}

func (n *OnlineNotify) Notify(userid int64) {
	idx := userid & chanmod

	n.chans[idx].Notify(userid)
}



