package main

import (
	"flag"
	"io/ioutil"
	"encoding/json"
)

var (
	Conf	*Config

	confFile string
)

func init() {
	flag.StringVar(&confFile, "c", "./client.conf", " set client config file path")
	Conf = new(Config)
	Conf.Protocol.Type = "ws"
	Conf.Protocol.Addr = "localhost:8090"
	Conf.Protocol.SndBuf = 256
	Conf.Protocol.RcvBuf = 256
	
	var (
		data []byte
		 err error
	)
	
	data, err = ioutil.ReadFile(confFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, Conf)
	if err != nil {
		return
	}
	
}

type ConfigUser struct {
	From int64
	To		int64
}

type ConfigProtocol struct {
	Type		string
	Addr		string
	SndBuf	int
	RcvBuf	int
}

type Config struct {
	Protocol	ConfigProtocol
	User		ConfigUser
}





