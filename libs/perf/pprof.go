package perf

import (
	"net/http"
	_ "net/http/pprof"

	log "github.com/ikenchina/golog"
)

// StartPprof start http pprof.
func Init(pprofBind []string)  {
//	pprofServeMux := http.NewServeMux()
//	pprofServeMux.HandleFunc("/debug/pprof/", pprof.Index)
//	pprofServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
//	pprofServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
//	pprofServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
//	pprofServeMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	for _, addr := range pprofBind {
		go func() {
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Fatalf("perf.Init failed : %v", err)
				panic(err)
			} else {
				log.Infof("perf listen addr : %v", addr)
			}
		}()
	}
}
