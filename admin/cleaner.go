package main

import (
	"sync"
	"time"
	"container/list"
	//log "github.com/ikenchina/golog"
)

const (
	maxCleanNum = 1000
)

type CleanData struct {
	Key        int64
	Server  int32
	expireTime time.Time
}
func (c *CleanData) expire() bool {
	return c.expireTime.Before(time.Now())
}

type Cleaner struct {
	sync.Mutex
	size  int
	dataList *list.List
	maps  map[int64]*list.Element
}

func NewCleaner(cleaner int) *Cleaner {
	c := new(Cleaner)
	c.dataList  = list.New()
	c.size = 0
	c.maps = make(map[int64]*list.Element, cleaner)
	return c
}

func (c *Cleaner) PushFront(user int64, server int32, expire time.Duration) {
	c.Lock()
	if e, ok := c.maps[user]; ok {
		// update time
		ee := e.Value.(*CleanData)
		ee.expireTime = time.Now().Add(expire)
		
		c.dataList.MoveToFront(e)
	} else {
		e := new(CleanData)
		e.Key = user
		e.Server = server
		e.expireTime = time.Now().Add(expire)
		
		ee := c.dataList.PushFront(e)
		c.maps[user] = ee
		
		c.size++
	}
	c.Unlock()
}



func (c *Cleaner) Remove(key int64) {
	c.Lock()

	c.remove(key)
	
	c.Unlock()
}

func (c *Cleaner) remove(key int64) {
	if e, ok := c.maps[key]; ok {
		delete(c.maps, key)
		c.dataList.Remove(e)
		c.size--
	}
}

func (c *Cleaner) Clean() (data []*CleanData) {

	data = make([]*CleanData, 0, maxCleanNum)
	c.Lock()

	var p *list.Element
	e := c.dataList.Back()
	for e != nil {
		p = e.Prev()
		vv := e.Value.(*CleanData)
		
		if vv.expire() {
			c.remove(vv.Key)
			data = append(data, vv)
		} else {
			break
		}

		e = p
	}

	c.Unlock()
	return
}

func (c *Cleaner) GetDataSize() (count int) {

	c.Lock()
	count = c.size
	c.Unlock()
	return
}

