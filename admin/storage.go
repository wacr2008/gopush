package main

import (
	"sync"
	"strconv"
	"gopkg.in/redis.v3"
	"github.com/stathat/consistent"
	"gopush/libs/proto"
	"github.com/pquerna/ffjson/ffjson"
	"gopush/libs/nocopy"
	log "github.com/ikenchina/golog"
)

// persist to storage for ensure message


type Storage interface {
	Init(conf ConfigStorage) error
	SaveMsg(msgs *proto.PushMsg) error
	SaveMMsg(userid []int64, msg *proto.PushMsg) ([]int64, error)
	DelMsg(msg *proto.PushMsg) error
	GetMsg(userid, start int64, stop int64) ([]*proto.PushMsg, error)
	Size(userid int64) int64
	UnInit()
	UpdateConfig()
}


func InitStorage() (Storage,error) {
	conf := Conf.Storage
	if conf.Type == "redis" {
		r := new(RedisStorage)
		err := r.Init(conf)
		return r, err
	}
	return nil, ErrUnknowStorage
}

func UnInitStorage() {
	DefaultStorage.UnInit()
}


type RedisStorage struct {
	sync.RWMutex
	redisCluster map[string]*redis.Client
	consistentHash *consistent.Consistent
	Config		ConfigStorage
	
	maxKey	string
}

func (r *RedisStorage) Init(conf ConfigStorage) error {
	r.redisCluster = make(map[string]*redis.Client, 0)
	r.consistentHash = consistent.New()
	r.Config = conf
	
	zz := int64(1) << 62
	r.maxKey = strconv.FormatInt(zz, 10)
	
	return r.initRedis()
}

func (r *RedisStorage) UnInit() {
	for _, cli := range r.redisCluster {
		cmd := cli.BgSave()
		if cmd.Err() != nil {
			log.Errorf("RedisStorage.UnInit failed : %v", cmd.Err())
		}
	}
}
func (r *RedisStorage) UpdateConfig() {

}

func (r *RedisStorage) initRedis() error {
	//utils.CatchError()
	r.Lock()
	defer r.Unlock()
	
	log.Infoln("RedisStorage.initRedis")
	
	for _, n := range r.consistentHash.Members() {
		r.consistentHash.Remove(n)
	}

	for _, node := range r.Config.Nodes {
		cli, err := r.newRedisClient(node)
		
		if err == nil && cli != nil {
			log.Infof("RedisStorage.initRedis node succesful (%s@%s)", node.Name, node.Addr)
			r.consistentHash.Add(node.Name)
			r.redisCluster[node.Name] = cli
		}
	}
	if len(r.redisCluster) == 0 {
		return ErrStorageInitError
	}
	return nil
}


func (r *RedisStorage) newRedisClient(node ConfigStorageNode) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:  node.Addr,
	})
	
	_, err := client.Ping().Result()
	if err != nil {
		log.Fatalf("%v\n", err)
		return nil, err
	}

	return client, nil
}



func (r *RedisStorage) SaveMsg(msg *proto.PushMsg) (error) {

	r.RLock()
	defer r.RUnlock()
	uid := strconv.FormatInt(msg.UserId, 10)
	redisnode, err := r.consistentHash.Get(uid)
	if err != nil {
		return err
	}
	client, ok := r.redisCluster[redisnode]
	if !ok || client == nil {
		return ErrStorageNodeNotExist
	}
	var data []byte
	data, err = ffjson.Marshal(msg)
	if err != nil {
		return err
	}
	
	// warning : maybe overflow
	client.ZAdd(uid, redis.Z{float64(msg.Id), data})
	
	return nil
}


func (r *RedisStorage) GetMsg(userid, start int64, stop int64) ([]*proto.PushMsg, error) {
	
	r.RLock()
	defer r.RUnlock()
	
	uid := strconv.FormatInt(userid, 10)
	redisnode, err := r.consistentHash.Get(uid)
	if err != nil {
		return nil, err
	}
	client, ok := r.redisCluster[redisnode]
	if !ok || client == nil {
		return nil, ErrStorageNodeNotExist
	}
	
	existdo := client.Exists(uid)
	if (!existdo.Val()) {
		return nil, nil
	}
	
	rangedo := client.ZRange(uid, start, stop)
	if rangedo.Err() != nil {
		return nil, rangedo.Err()
	}
	//log.Debugln((rangedo.Val()), uid, start, stop)
	
	msgs := make([]*proto.PushMsg,0)
	for _, vv := range rangedo.Val() {
		bb := nocopy.Slice(vv)
		m := new(proto.PushMsg)
		err = ffjson.Unmarshal(bb, m)
		if err == nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func (r *RedisStorage) Size(userid int64) int64 {
	r.RLock()
	defer r.RUnlock()
	
	uid := strconv.FormatInt(userid, 10)
	redisnode, err := r.consistentHash.Get(uid)
	if err != nil {
		return 0
	}
	client, ok := r.redisCluster[redisnode]
	if !ok || client == nil {
		return 0
	}
	
	do := client.ZCount(uid, "0", r.maxKey)
	if do.Err() != nil {
		return 0
	}
	return do.Val()
	
}

func (p *RedisStorage) DelMsg(msg *proto.PushMsg) error {

	p.RLock()
	defer p.RUnlock()
	
	user := msg.UserId
	id := msg.Id
	uid := strconv.FormatInt(user, 10)
	
	redisnode, err := p.consistentHash.Get(uid)
	if err != nil {
		log.Errorf("RedisStorage.DelMsg(%d, %d) failed : %v", user, id, err)
		return err
	}
	client, ok := p.redisCluster[redisnode]
	if !ok || client == nil {
		log.Errorf("RedisStorage.DelMsg failed : redis client is nil")
		return ErrStorageNodeNotExist
	}
	
	min := strconv.FormatInt(id, 10)
	
	cmd := client.ZRemRangeByScore(uid, min, min)
	if cmd.Err() != nil {
		log.Errorln(cmd.Err())
	}
	return cmd.Err()
}


//  
// @TODO :	1. optimize
//			2. transactions
func (r *RedisStorage) SaveMMsg(userid []int64, msg *proto.PushMsg) ([]int64, error) {
	r.RLock()
	defer r.RUnlock()
	
	var err error
	failedUsers := make([]int64, 0)
	for _, u := range userid {
		msg.Id = GenerateId()
		msg.UserId = u
		err = r.SaveMsg(msg)
		if err != nil {
			failedUsers = append(failedUsers, u)
		}
	}
	return failedUsers, nil
	
	/*
	failedUsers := make([]int64)
	redismap := make(map[*redis.Client][]int64)
	for _, u := range userid {
		uid := strconv.FormatInt(u, 10)
		redisnode, err := r.consistentHash.Get(uid)
		if err != nil {
			failedUsers = append(failedUsers, u)
		}
		client, ok := r.redisCluster[redisnode]
		if !ok || client == nil {
			failedUsers = append(failedUsers, u)
		} else {
			redismap[client] = append(redismap[client], u)
		}
	}
	
var data []byte
	for c, us := range redismap {
		md := c.Multi()
		for _, u := range us {
			uid := strconv.FormatInt(u, 10)
			var data []byte
			msg.Id = GenerateId()
			msg.UserId = u

			data, err = ffjson.Marshal(msg)
			if err != nil {
				return
			}
			md.ZAdd(uid, redis.Z{msg.Id, data})
			
		}

	}
	*/
	
	
	
	
	return failedUsers, nil
}




