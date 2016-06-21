package main

import (
	"flag"
	"runtime"
	"fmt"
	log "github.com/ikenchina/golog"
	"gopush/libs/signal"
)

var(
	isDebug bool
	cometRPC *CometRpc
	MsgStorage Storage
)

func main() {
	flag.Parse()
	var err error
	fmt.Println("init configuration")
	if err = InitConfig(); err != nil {
		panic(err)
	}

	isDebug = Conf.Common.Debug
	runtime.GOMAXPROCS(Conf.Common.MaxProc)
	
	// set log
	if err := InitLog(Conf.Log); err != nil {
		fmt.Printf("init log failed : %v", err)
	}
	defer log.Close()
	log.Infof("init log successful")

	//comet
	cometRPC = NewCometRpc()
	if err = cometRPC.Init(); err != nil {
		log.Errorf("init comet failed : %v", err)
		panic(err)
	}

	mq, err := InitMQ()
	if err != nil {
		log.Errorf("init mq failed : %v", err)
		panic(err)
	}
	
	err = InitRPC()
	if err != nil {
		log.Errorf("init rpc failed : %v", err)
		panic(err)
	}
	MsgStorage, err = InitStorage()
	if err != nil {
		log.Errorf("init storage failed : %v", err)
		panic(err)
	}
	
	Conf.RegisterUpdateNotify(cometRPC.UpdateConfig)
	
	signal.RegisterExit(cometRPC.UnInit)
	signal.RegisterExit(mq.UnInit)
	signal.RegisterReload(reload)
	signal.WaitSignal()
}

func reload() {
	err := ReloadConfig()
	if err != nil {
		log.Errorf("ReloadConfig() error(%v)", err)
	}
}
