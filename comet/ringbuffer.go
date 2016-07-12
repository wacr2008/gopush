package main

import (
	"github.com/ikenchina/gopush/libs/proto"
)

// ref : 

type RingBuffer struct {
	rp   uint64		// read point
	num  uint64	
	mask uint64
	// @TODO cacheline, most cache line size is 64, not all
	cacheLinePadding [40]byte

	wp   uint64	// write point
	data []proto.Proto
}

func NewRing(num int) *RingBuffer {
	r := new(RingBuffer)
	r.init(uint64(num))
	return r
}

func (r *RingBuffer) Init(num int) {
	r.init(uint64(num))
}

func (r *RingBuffer) init(num uint64) {
	// 2^N
	if num&(num-1) != 0 {
		for num&(num-1) != 0 {
			num &= (num - 1)
		}
		num = num << 1
	}
	r.data = make([]proto.Proto, num)
	r.num = num
	r.mask = r.num - 1
	r.cacheLinePadding = [40]byte{}
}

func (r *RingBuffer) Get() (proto *proto.Proto, err error) {
	if r.rp == r.wp {
		return nil, ErrRingBufferEmpty
	}
	proto = &r.data[r.rp&r.mask]
	return
}

func (r *RingBuffer) GetAdv() {
	r.rp++
}

func (r *RingBuffer) Set() (proto *proto.Proto, err error) {
	if r.wp-r.rp >= r.num {
		return nil, ErrRingBufferFull
	}
	proto = &r.data[r.wp&r.mask]
	return
}

func (r *RingBuffer) SetAdv() {
	r.wp++
}

func (r *RingBuffer) Reset() {
	r.rp = 0
	r.wp = 0
}
