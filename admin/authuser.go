package main


// auth userid match token
type UserAuther interface {
	Auth(userid int64, token string) bool
}

type DefaultAuther struct {
}

func NewDefaultAuther() *DefaultAuther {
	return &DefaultAuther{}
}

// 
func (a *DefaultAuther) Auth(userid int64, token string) bool {
	return true
}

//type RedisAuther struct {
	
//}