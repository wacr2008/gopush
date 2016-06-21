package main

import (
	"sync"
	"time"
	//log "github.com/ikenchina/golog"
)

type Session struct {
	Topics	[]int32
	ServerId	int32
}

type ServerSession struct {
	userSession	map[int64]*Session
	topicCount	map[int32]int64
}

func NewServerSession() *ServerSession {
	s := new(ServerSession)
	s.userSession = make(map[int64]*Session)
	s.topicCount = make(map[int32]int64)
	return s
}

// NewSession new a session for a user
func NewSession(server int32, topics []int32) *Session {
	s := new(Session)
	s.ServerId = server
	s.Topics = topics
	return s
}

type Bucket struct {
	sync.RWMutex
	topicServers	map[int32]map[int32]struct{}	// map[topic]map[serverid]struct{}
	serverSession	map[int32]*ServerSession
	userServer	map[int64]int32			// userid -> serverid
	topicCount	map[int32]int64			//  map[topic]userCount
	
	cleaner		*Cleaner			// bucket map cleaner
	Config		ConfigBucket
}

// NewBucket 
func NewBucket(conf ConfigBucket) *Bucket {
	b := new(Bucket)
	b.topicServers = make(map[int32]map[int32]struct{})
	b.topicCount =  make(map[int32]int64)
	b.serverSession = make(map[int32]*ServerSession)
	b.userServer = make(map[int64]int32)
	
	b.Config = conf
	b.cleaner = NewCleaner(b.Config.Cleaner)
	
	go b.clean()
	return b
}

//  incr or decr counter.
func (b *Bucket) counter(userId int64, server int32, topics []int32, incr bool) {
	if incr {
		 for _, topic := range topics {
		 	b.topicCount[topic] = b.topicCount[topic] + 1
		 	b.serverSession[server].topicCount[topic] = b.serverSession[server].topicCount[topic] + 1
		 }
	} else {
		var count int64
		for _, topic := range topics {
			if b.topicCount[topic] > 0 {
				b.topicCount[topic] = b.topicCount[topic] - 1
			}
			count = b.serverSession[server].topicCount[topic]
			if  count > 0 {
				b.serverSession[server].topicCount[topic] = count - 1
			}
		}
	}
}

func (b *Bucket) GetTopicServers(topic int32) []int32 {
	var servers []int32
	b.RLock()

	if mm, ok := b.topicServers[topic]; ok {
		for s,_  := range mm {
			servers = append(servers, s)
		}
	} 
	
	b.RUnlock()
	return servers
}

// Put put a channel according with user id.
func (b *Bucket) Put(userId int64, server int32, topics []int32) error {
	
	var (
		si *ServerSession
		s *Session
		ok bool
	)

	b.Lock()
	defer b.Unlock()
	
	if si, ok = b.serverSession[server]; !ok {
		si = NewServerSession()
		b.serverSession[server] = si
	}
	s, ok = si.userSession[userId]
	if !ok {
		s = NewSession(server, topics)
		si.userSession[userId] = s
	}
	//log.Debugln(userId)
	// remove userid from cleaner first if it exist 
	// avoid put->del->put , cleaner delete user who last put
	b.cleaner.Remove(userId)
	b.userServer[userId]=server

	for _, topic := range topics {
		if s, ok := b.topicServers[topic]; !ok {
			s = make(map[int32]struct{})
			b.topicServers[topic] = s
		}
		if _, ok := b.topicServers[topic][server]; !ok {
			//log.Debugln(topic, server)
			b.topicServers[topic][server] = struct{}{}
		}
	}
	b.counter(userId, server, topics, true)
	
	//log.Debugf("put %d  to %d", userId, server)
	
	
	return nil
}

// Del delete the channel
func (b *Bucket) Del(userId int64, server int32) error {
	var (
		si *ServerSession
		s *Session
		ok bool
	)
	//log.Debugln(userId)
	b.Lock()
	defer b.Unlock()
	
	if si, ok = b.serverSession[server]; !ok {
		return ErrServerNotExist
	}
	if s, ok = si.userSession[userId]; !ok {
		return ErrUserNotExist
	}
	b.counter(userId, server, s.Topics, false)
	delete(b.userServer, userId)
	
	for _, t := range s.Topics {
		if si.topicCount[t] == 0 {
			if _, ok = b.topicServers[t][server]; ok {
				delete(b.topicServers[t], server)
			}
		}
	}
	
	b.cleaner.PushFront(userId, server, time.Duration(b.Config.Session.Expire))
	
	return nil
}

func (b *Bucket) GetUserServer(userid int64) (server int32, err error) {
	server = -1
	err = ErrServerNotExist
	var ok bool
	
	b.RLock()
	if server, ok = b.userServer[userid]; ok {
		err = nil
	}
	b.RUnlock()
	return
}

func (b *Bucket) DelServer(server int32) error {
	var si *ServerSession
	var ok bool
	
	b.Lock()

	if si, ok = b.serverSession[server]; !ok {
		b.Unlock()
		return ErrServerNotExist
	}
	
	delete(b.serverSession, server)
	
	for user, _ := range si.userSession {
		delete(b.userServer, user)
	}
	for t,_ := range si.topicCount {
		if _, ok := b.topicServers[t]; ok {
			delete(b.topicServers[t], server)
		}
	}
	for t, c := range si.topicCount {
		if tc, ok := b.topicCount[t]; ok {
			b.topicCount[t] = tc - c
		}
	}
	b.Unlock()
	return nil
}


func (b *Bucket) delEmpty(userId int64, serverId int32) {
	//if isDebug {
	//		log.Debugf("delEmpty : user(%d)  server(%d)", userId, serverId)
	//}
	if si, ok := b.serverSession[serverId]; ok {
		delete(si.userSession, userId)
	}
}

func (b *Bucket) clean() {
	var (
		i       int
		data []*CleanData
	)
	for {
		b.Lock()
		data = b.cleaner.Clean()
		//log.Debugf("%v  -->  %v", len(b.userServer), len(data))
		if len(data) != 0 {
			for i = 0; i < len(data); i++ {
				b.delEmpty(data[i].Key, data[i].Server)
			}
		}
		b.Unlock()
		time.Sleep(time.Duration(b.Config.CleanPeriod))
	}
}

func (b *Bucket) GetServerSize(serverid int32) (size int64) {
	b.RLock()
	
	if ss, ok := b.serverSession[serverid]; ok {
		size = int64(len(ss.userSession))
	}
	
	b.RUnlock()
	return
}

func (b *Bucket) GetUsers(serverid int32) []*User {
	users := make([]*User, 0)
	b.RLock()
	
	if se, ok := b.serverSession[serverid]; ok {
		for user, s := range se.userSession {
			u := &User{user, s.Topics}
			users = append(users, u)
		}
	}
	
	b.RUnlock()
	
	return users
}


func (b *Bucket) GetCleanDataSize() int {
	
	return b.cleaner.GetDataSize()
	
}

func (b *Bucket) GetServers() []int32 {
	ss := make([]int32, 0)
	b.RLock()
	
	for s, _ := range b.serverSession {
		ss = append(ss, s)
	}
	
	b.RUnlock()
	
	return ss
}

type User struct {
	Id	int64
	Topics []int32
}
	
	