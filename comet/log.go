package main

import(
	"os"
	"time"
	log "github.com/ikenchina/golog"
)

func InitLog(conf ConfigLogs) error {
	
	log.SetLevel(conf.Level)
	var h log.Handler
	var err error
	for _, l := range conf.Handlers {
		switch l.Type {
			case "stdout":
				h, err = log.NewStreamHandler(os.Stdout)
			case "sentry":
				h, err = log.NewSentryHandler(l.SentryDSN)
			case "socket":
				h, err = log.NewSocketHandler(l.SocketProtocol, l.SocketAddr, time.Duration(l.SocketTimeout))
			case "sizeRotateFile":
				h, err = log.NewSizeRotateFileHandler(l.FileName, l.FileSize, l.FileCount)
		}
		if err == nil {
			log.AppendHandler(h)
		} else {
			return err
		}
	}
	return err
}

