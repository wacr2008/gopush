package main

import (
	
	//log "github.com/ikenchina/golog"
)
type Server struct {
	
	Buckets		[]*Bucket
	BucketSize	int64

	Auther 		UserAuther
	Config		ConfigServer
	
	servers		map[int32]struct{}
}

// NewServer returns a new Server.
func NewServer() (*Server, error) {
	s := new(Server)
	return s, nil
}

func (s *Server) Init() error {
	
	s.Config = Conf.Server
	s.Auther = NewDefaultAuther()
	
	s.Buckets = make([]*Bucket, s.Config.Bucket.Size)
	for i := 0; i < s.Config.Bucket.Size; i++ {
		s.Buckets[i] = NewBucket(s.Config.Bucket)
	}
	
	s.BucketSize = int64(s.Config.Bucket.Size)
	
	return nil
}

func (s *Server) UnInit() {
	
}

func (s *Server) Sub(userid int64, serverid int32, topics []int32) error {

	return s.GetBucket(userid).Put(userid, serverid, topics)
	
}

func (s *Server) UnSub(userid int64, serverid int32) error {
	return s.GetBucket(userid).Del(userid, serverid)
}

func (s *Server) GetBucket(userid int64) *Bucket {
	idx := userid % s.BucketSize
	return s.Buckets[idx]
}

func (s *Server) GetTopicServers(topic int32) []int32 {
	var servers []int32
	smap := make(map[int32]struct{})
	for _, b := range s.Buckets {
		for _, s := range b.GetTopicServers(topic) {
			smap[s] = struct{}{}
		}
	}
	for s, _ := range smap{
		servers = append(servers, s)
	}
	return servers
}

func (s *Server) DeleteServer(serverId int32) error {
	var err error
	for _, b := range s.Buckets {
		err = b.DelServer(serverId)
		if err != nil && err != ErrServerNotExist {
			return err
		}
	}
	return nil
}

func (s *Server) GetServerSize(serverId int32) (count int64) {
	
	for _, bk := range s.Buckets {
		count += bk.GetServerSize(serverId)
	}
	return
}

func (s *Server) GetServerSession() map[int32][]*ServerSession {
	
	serverSessions := make(map[int32][]*ServerSession)
	for _, bk := range s.Buckets {
		for sid, sinfo := range bk.serverSession {
			serverSessions[sid] = append(serverSessions[sid], sinfo)
		}
	}
	
	return serverSessions
}

func (s *Server) GetUsers(serverid int32) []*User {
	users := make([]*User, 0)
	for _, bk := range s.Buckets {
		
		us := bk.GetUsers(serverid)
		users = append(users, us...)
		
	}
	return users
}


func (s *Server) GetServers() []int32 {
	
	smaps := make(map[int32]struct{}, 0)
	
	for _, bk := range s.Buckets {
		svrs := bk.GetServers()
		for _, ss := range svrs {
			if _, ok := smaps[ss]; !ok {
				smaps[ss] = struct{}{}
			}
		}
	}
	ret := make([]int32, 0)
	for ss, _ := range smaps {
		ret = append(ret, ss)
	}
	
	return ret
}


