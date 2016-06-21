package main

import (
	"fmt"
	"bytes"
	"io/ioutil"
	"net/http"
	"log"
	"time"
)



func pushTopic(topic int32) {
	
	
	msg :=`{"topic" : 1,
	"msg":{
		"body" : "yyy"	
	}}`
	
	body := bytes.NewBuffer([]byte(msg))
	client := &http.Client{}
	resp, err := client.Post("http://localhost:8880/push/topic", "application/json;charset=utf-8", body)
	if err != nil {
		log.Println(err)
	}
	ret, err := ioutil.ReadAll(resp.Body)
	log.Println(string(ret))
}

func pushUser(user int64) {
	t :=`{
	"userid" : %d,
	"ensure" : true,
	"expire" : 0,
	"redomax" : -1,
	"body" : "{xx}"	
}`
msg := fmt.Sprintf(t, user)
	
	body := bytes.NewBuffer([]byte(msg))
	client := &http.Client{}
	resp, err := client.Post("http://localhost:8880/push/user", "application/json;charset=utf-8", body)
	if err != nil {
		log.Println(err)
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	st := time.Now()
	
	for i:=0 ;i < 10000000; i++ {
		pushUser(1)
	}
	
	et := time.Now()
	log.Println(et.Sub(st))
}




