package main

import (
	"sync"
	"strconv"
	"gopkg.in/redis.v3"
	"github.com/stathat/consistent"
log "github.com/ikenchina/golog"
)

// 持久化消息到存储中


type Storage interface {
	Init(conf ConfigStorage) error
	DelMsg(user int64, id int64) error
}


func InitStorage() (Storage, error) {
	if Conf.Storage.Type == "redis" {
		r := new(RedisStorage)
		err := r.Init(Conf.Storage)
		return r, err
	}
	return nil, ErrUnknowStorage
}


type RedisStorage struct {
	sync.RWMutex
	redisCluster map[string]*redis.Client
	consistentHash *consistent.Consistent
	Config		ConfigStorage
}

func (r *RedisStorage) Init(conf ConfigStorage) error {
	r.redisCluster = make(map[string]*redis.Client, 0)
	r.consistentHash = consistent.New()
	r.Config = conf
	return r.initRedis()
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


func (p *RedisStorage) newRedisClient(node ConfigStorageNode) (*redis.Client, error) {
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


func (p *RedisStorage) DelMsg(user int64, id int64) error {

	p.RLock()
	defer p.RUnlock()
	
	
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






