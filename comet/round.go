package main

import (
	"gopush/libs/bytes"
	"gopush/libs/time"
)

type RoundRobinOptions struct {
	Timer			int
	TimerSize			int
	Reader			int
	ReadBuf			int
	ReadBufSize		int
	Writer			int
	WriteBuf			int
	WriteBufSize		int
}

// Round-robin for timer, reader, writer
type RoundRobin struct {
	readers		[]bytes.Pool	// array pool : gc faster
	writers		[]bytes.Pool
	timers		[]time.Timer
	
	readerIdx		int
	writerIdx		int
	timerIdx		int
	
	Config	ConfigRoundRobin
}


func NewRoundRobin(conf ConfigRoundRobin) (r *RoundRobin) {
	var i int
	r = new(RoundRobin)
	r.Config = conf
	// reader
	r.readers = make([]bytes.Pool, conf.Reader)
	for i = 0; i < conf.Reader; i++ {
		r.readers[i].Init(conf.ReadBuf, conf.ReadBufSize)
	}
	// writer
	r.writers = make([]bytes.Pool, conf.Writer)
	for i = 0; i < conf.Writer; i++ {
		r.writers[i].Init(conf.WriteBuf, conf.WriteBufSize)
	}
	// timer
	r.timers = make([]time.Timer, conf.TimerNum)
	for i = 0; i < conf.TimerNum; i++ {
		r.timers[i].Init(conf.TimerSize)
	}
	return
}

// Timer get a timer.
func (r *RoundRobin) Timer(rn int) *time.Timer {
	return &(r.timers[rn%r.Config.TimerNum])
}

// Reader get a reader memory buffer.
func (r *RoundRobin) Reader(rn int) *bytes.Pool {
	return &(r.readers[rn%r.Config.Reader])
}

// Writer get a writer memory buffer pool.
func (r *RoundRobin) Writer(rn int) *bytes.Pool {
	return &(r.writers[rn%r.Config.Writer])
}
