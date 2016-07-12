package main

import (
	"flag"
	"runtime"
	"fmt"
	
	log "github.com/ikenchina/golog"
	"github.com/ikenchina/gopush/libs/signal"
	"github.com/ikenchina/gopush/libs/perf"
)


var (
	CometServer *Server
	isDebug         bool
)


func main() {
	flag.Parse()
	
	// init config
	if err := InitConfig(); err != nil {
		log.Fatalf("InitConfig failed : %v", err)
		panic(err)
	}
	fmt.Println("init configure successful")
	
	isDebug = Conf.Common.Debug
	runtime.GOMAXPROCS(Conf.Common.MaxProc)
	
	// set log 
	if err := InitLog(Conf.Log); err != nil {
		log.Fatalf("init log failed : %v", err)
		fmt.Printf("init log failed : %v", err)
		panic(err)
	}
	defer log.Close()
	log.Infof("init log successful")
	
	// init pprof
	perf.Init(Conf.Common.PprofBind)
	
	//  server
	CometServer = NewServer()
	if err := CometServer.Init(); err != nil {
		log.Fatalf("CometServer.Init failed : %v", err)
		panic(err)
	}
	log.Infof("init comet server successful")
	
	//  rpc server
	if err := InitRPC(); err != nil {
		log.Fatalf("InitRpcServer failed : %v", err)
		panic(err)
	}
	log.Infof("init  rpc  server successful")

	Conf.RegisterUpdateNotify(CometServer.UpdateConfig)
	Conf.RegisterUpdateNotify(UpdateRpcConfig)
	
	// wait signal
	signal.RegisterExit(CometServer.UnInit)
	signal.RegisterExit(UnInitRPC)
	signal.RegisterReload(reload)
	signal.WaitSignal()
}

func reload() {
	err := ReloadConfig()
	if err != nil {
		log.Errorf("ReloadConfig failed : %v", err)
		return
	}
}
