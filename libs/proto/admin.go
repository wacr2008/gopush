package proto


// if you modify it . please run ffjson admin.go to update

// subscribe request struct
type SubREQ struct {
	UserId	int64
	Token	string
	Topics	[]int32
	Server	int32
}

// subscribe response struct
type SubRES struct {
	Pass		bool
}

// unsubscribe request struct
type UnSubREQ struct {
	UserId	int64
	Server	int32
}

// unsubscribe response struct
type UnSubRES struct {
	Has		bool
}