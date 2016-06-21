package main


import (
	"time"
)


// @TODO  optimize : score of redis sorted set is float64, so need to divided by 100
// 
func GenerateId() int64 {
	return time.Now().UnixNano() / 100
}



