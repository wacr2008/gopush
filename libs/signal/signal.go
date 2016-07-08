package signal

import (
	"os"
	"os/signal"
	"syscall"
	"container/list"
)

type Func func()

type signalData struct {
	ExitFuncs *list.List
	ReloadFuncs *list.List
}

var datas *signalData

func init() {
	datas = new(signalData)
	datas.ExitFuncs = list.New()
	datas.ReloadFuncs = list.New()
}


func WaitSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			for e := datas.ExitFuncs.Front(); e != nil ; e = e.Next() {
				e.Value.(Func)()
			}
			os.Exit(0)
			return
		case syscall.SIGHUP:
			for e := datas.ReloadFuncs.Front(); e != nil ; e = e.Next() {
				e.Value.(Func)()
			}
		default:
			return
		}
	}
}

func RegisterReload(f Func) {
	datas.ReloadFuncs.PushBack(f)
}
func RegisterExit(f Func) {
	datas.ExitFuncs.PushBack(f)
}

