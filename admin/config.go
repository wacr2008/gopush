package main

import (
	"flag"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

var (
	Conf     *Config
	confFile string
)

func init() {
	flag.StringVar(&confFile, "c", "./admin.conf", " set comet config file path")
}


type ConfigCommon struct {
	Debug	bool
	MaxProc	int
	PprofBind	[]string
	Dir			string
}
type ConfigLog struct {
	Type		string
	SocketProtocol	string
	SocketAddr	string
	SocketTimeout	int64
	SentryDSN	string
	FileName		string
	FileSize		int64
	FileCount		int
	
}
type ConfigLogs struct {
	Level	int
	Handlers		[]ConfigLog
}

type ConfigBindAddr struct {
	Network	string
	Addr		string
}

type ConfigRPC struct {
	Bind		[]ConfigBindAddr
}

// @TODO timeout should be duration, not second
type ConfigHttp struct {
	Bind		[]ConfigBindAddr
	
}
type ConfigMQ struct {
	Type		string
	Addrs	[]string
}

//暂时只做redis，不定义其他值了
type ConfigStorageNode struct {
	Name	string
	Addr		string
	User		string
	Password	string
}
type ConfigStorage struct {
	Type		string
	Nodes	[]ConfigStorageNode
}


type ConfigSession struct {
	Expire	int64
}
type ConfigBucket struct {
	Size	int
	CleanPeriod	int64
	Cleaner	int
	Session	ConfigSession
}


type ConfigServer struct {
	Id		int
	Bucket	ConfigBucket
}

type updateFunc func()
type Config struct {
	Common		ConfigCommon
	Log			ConfigLogs
	RPC			ConfigRPC
	Http			ConfigHttp
	Server		ConfigServer
	MQ			ConfigMQ
	Storage		ConfigStorage
	UpdateFuncs	[]updateFunc
}

// @TODO check the necessary configure
func (c *Config) IsValid() bool {
	return true
}

func InitConfig() (error) {
	return ReloadConfig()
}

func ParseConfigFile() (conf *Config, err error) {
	var (
		data []byte
	)
	
	data, err = ioutil.ReadFile(confFile)
	if err != nil {
		return
	}
	conf = new(Config)
	err = yaml.Unmarshal(data, conf)
	if err != nil {
		return
	}
	return
}


// reload configure from :
//  1. from file
//  2. from configure centre : @TODO : from configure centre (zk, etcd...)
func ReloadConfig() (error) {
	conf, err := ParseConfigFile()
	if err == nil {
		Conf = conf
		for _, f := range Conf.UpdateFuncs {
			f()
		}
	}
	return err
}

// register notify function of configuration update
func (c *Config)RegisterUpdateNotify(fun func()) {
	c.UpdateFuncs = append(c.UpdateFuncs, updateFunc(fun))
}



