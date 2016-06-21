package main

import (
	"time"
	"gopush/libs/proto"
	//log "github.com/ikenchina/golog"
)

func PushMsg(serverid int32, msg *proto.PushMsg) error {
	msg.Id = GenerateId()
	msg.Time = time.Now().Unix()

	if msg.Ensure {
		err := DefaultStorage.SaveMsg(msg)
		
		if err != nil {
			return err
		}
	}
	err := DefaultMQ.Push(serverid, msg)
	if err != nil {
		if !msg.Ensure {
			return err
		}
	}
	return nil
}

func MPushMsg(sid int32, uids []int64, msg *proto.PushMsg) error {
	

	
//	err := DefaultStorage.Save(msg)
//	if err != nil {
//		return err
//	}
	
//	err = DefaultMQ.MPush(sid, uids, msg)
//	if err != nil {
//		log.Errorf("push to message queue failed : %v", err)
//	}
	return nil
}
func SaveMsg(msg *proto.PushMsg) error {

	msg.Id = GenerateId()
	msg.Time = time.Now().Unix()
	
	return DefaultStorage.SaveMsg(msg)
	
}
