package main

import (
	"flag"
	"os"
	"time"
	"strconv"
	log "github.com/ikenchina/golog"
)

func main() {
	flag.Parse()
	
	h, _ := log.NewStreamHandler(os.Stdout)
	log.SetLevel(log.LevelDebug)
	log.AppendHandler(h)
	defer log.Close()
	
	var(
		start int64
		end int64
	)
	if len(os.Args) > 2 {
		start,_ = strconv.ParseInt(os.Args[1], 10, 0)
		end,_ = strconv.ParseInt(os.Args[2], 10, 0)
	} else {
		start = Conf.User.From
		end = Conf.User.To
	}
	
	log.Infoln(time.Now())
	if Conf.Protocol.Type == "tcp" {
		initTCP()
	} else if Conf.Protocol.Type == "ws" {
		var i int64
		for i= start; i < end; i++ {
			go initWebsocket(i, []int32{1,2,3})
		}
		log.Infof("waiting")
		initWebsocket(end, []int32{1,2,3})
		
		
		
	} else if Conf.Protocol.Type == "wsssl" {
		//initWebsocketTLS()
	}
}
