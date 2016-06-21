package define

const (
	OP_INIT		=	int32(0)
	// handshake
	OP_HANDSHAKE_REQ	=	int32(1)
	OP_HANDSHAKE_RES	=	int32(2)
	
	// heartbeat
	OP_HEARTBEAT_REQ	=	int32(3)
	OP_HEARTBEAT_RES		=	int32(4)
	
	//subscribe
	OP_SUB_REQ		=	int32(5)
	OP_SUB_RES		=	int32(6)
	
	// unsubscribe
	OP_UNSUB_REQ	=	int32(7)
	OP_UNSUB_RES	=	int32(8)
	
	// raw message
	OP_RAW_MSG		=	int32(11)
	
	OP_MSG_RES		=	int32(12)
	
	// proto
	OP_PROTO_READY	=	int32(13)
	OP_PROTO_FINISH	=	int32(14)

)
