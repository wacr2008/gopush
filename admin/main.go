package main

import (
	"flag"
	"runtime"
	"fmt"
	log "github.com/ikenchina/golog"
	"gopush/libs/perf"
	"gopush/libs/signal"
)

var(
	DefaultServer	*Server
	DefaultMQ	MessageQueue
	DefaultStorage Storage
	isDebug		bool
)

func main() {
	flag.Parse()
	var err error
	if err = InitConfig(); err != nil {
		fmt.Sprintf("init configuration failed : %v", err)
		panic(err)
	}
	fmt.Println("init config successful.")
	
	isDebug = Conf.Common.Debug
	runtime.GOMAXPROCS(Conf.Common.MaxProc)
	
	// set log
	if err := InitLog(); err != nil {
		log.Fatalf("init log failed : %v", err)
		fmt.Printf("init log failed : %v", err)
		panic(err)
	}
	defer log.Close()
	fmt.Println("init log successful : ", Conf.Log)
	
	// init pprof
	perf.Init(Conf.Common.PprofBind)
	
	// init message queue
	DefaultMQ, err = InitMQ()
	if err != nil { 
		log.Fatalf("init message queue failed : %v", err)
		panic(err)
	}
	log.Infof("init messege queue successful.")
	
	// init storage
	DefaultStorage, err = InitStorage()
	if err != nil { 
		log.Fatalf("init storage failed : %v", err)
		panic(err)
	}
	log.Infof("init storage successful.")
	
	
	// init server
	DefaultServer, err = NewServer()
	err = DefaultServer.Init()
	if err != nil {
		log.Fatalf("init server failed : %v", err)
		panic(err)
	}
	
	// init admin rpc
	if err = InitRpc(); err != nil {
		log.Fatalf("init admin rpc failed : %v", err)
		panic(err)
	}
	
	// init http server
	if err = InitHTTP(); err != nil {
		log.Fatalf("init http server failed : %v", err)
		panic(err)
	}
	
	signal.RegisterExit(UnInitHTTP)
	signal.RegisterExit(UnInitRpc)
	signal.RegisterExit(DefaultMQ.UnInit)
	signal.RegisterExit(DefaultServer.UnInit)
	signal.RegisterExit(UnInitStorage)
	signal.RegisterExit(UnInitLog)
	
	signal.RegisterReload(reload)
	signal.WaitSignal()
	
}

func reload() {
	err := ReloadConfig()
	if err != nil {
		log.Errorf("ReloadConfig() error(%v)", err)
	}
}

