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
	flag.StringVar(&confFile, "c", "./comet.conf", " set comet config file path")
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

type ConfigChannel struct {
	Size				int
	RingBufferSize		int
	PushBufferSize	int
}
type ConfigBucket struct {
	Size		int
	Channel	ConfigChannel
}

type ConfigTcp struct {
	Bind			[]string
	SndBuf	int
	RcvBuf	int
	KeepAlive	bool
	
}
type ConfigWebSocket struct {
	Bind		[]string
	Tls		bool
	TlsCertFile	string
	TlsPrivateFile	string
}

type ConfigRoundRobin struct {
	TimerNum	int
	TimerSize		int
	Reader		int
	ReadBuf		int
	ReadBufSize	int
	Writer		int
	WriteBuf		int
	WriteBufSize	int
}


type ConfigServer struct {
	Id			int32
	HeartBeatTimeout	int64	
	Bucket			ConfigBucket
	RoundRobin		ConfigRoundRobin
	Tcp			ConfigTcp
	WebSocket		ConfigWebSocket
	RpcPush		[]ConfigRpcPush
	RpcAdmin		[]ConfigRpcAdmin
}	

type ConfigRpc struct {
	Network	string
	Addr		string
}
type ConfigRpcAdmin struct {
	Name	string
	Network	string
	Addr		string
}
type ConfigRpcPush struct {
	Name	string
	Network	string
	Addr		string
}

type updateFunc func()
type Config struct {
	Common	ConfigCommon
	Log		ConfigLogs
	Server		ConfigServer
	Rpc		[]ConfigRpc
	
	UpdateFuncs	[]updateFunc
}

// @TODO check the necessary configure
func (c *Config) IsValid() bool {
	if len(c.Common.PprofBind) == 0 || len(c.Server.WebSocket.Bind) == 0 || len(c.Rpc) == 0 || len(c.Server.RpcAdmin) == 0 {
		return false
	}
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
	if !conf.IsValid() {
		err = ErrInvalidConfig
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




