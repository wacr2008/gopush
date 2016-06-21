package proto

import (
	"encoding/json"
	"errors"
	"time"
	"golang.org/x/net/websocket"
	"gopush/libs/bufio"
	"gopush/libs/define"
	"gopush/libs/encoding/binary"
)


const (
	MaxBodySize = int32(1 << 10)
)

const (

	PackSize		=	4
	HeaderSize	=	2
	VerSize		=	2
	OperationSize	=	4
	MsgIdSize		=	8
	RawHeaderSize	=	PackSize + HeaderSize + VerSize + OperationSize + MsgIdSize
	MaxPackSize	=	MaxBodySize + int32(RawHeaderSize)
	EnsureSize	=	1
	//ExpireSize	=	4

	PackOffset	=	0
	HeaderOffset	=	PackOffset + PackSize
	VerOffset		=	HeaderOffset + HeaderSize
	OperationOffset	=	VerOffset + VerSize
	MsgIdOffset	=	OperationOffset + OperationSize
	EnsureOffset	=	MsgIdOffset + MsgIdSize
	//ExpireOffset	=	EnsureOffset + EnsureSize
)

var (
	emptyProto    = Proto{}
	emptyJSONBody = []byte("{}")

	ErrProtoPackLen   = errors.New("server protocol pack length error")
	ErrProtoHeaderLen = errors.New("server protocol header length error")
)

var (
	ProtocolVersion = int16(1)
	ProtoReady  = &Proto{OpCode: define.OP_PROTO_READY}
	ProtoFinish = &Proto{OpCode: define.OP_PROTO_FINISH}
)


// Proto is a request&response written before every connect.  It is used internally
// but documented here as an aid to debugging, such as when analyzing
// network traffic.
// tcp:
// binary codec
// websocket & http:
// raw codec, with http header stored ver, operation, seqid

// proto is protocol of communicate,
// 
type Proto struct {
	HeaderLen	int16			`json:"-"`		// header length
	Ver			int16			`json:"ver"`	// protocol version
	OpCode		int32			`json:"op"`	// operate code : define package
	MsgId		int64			`json:"msgid"`	// message id
	UserId		int64			`json:"-"`
	Ensure		bool			`json:"ensure"`
	//Expire		int32			`json:"expire"`
	Body		json.RawMessage	`json:"body"`	// binary body bytes(json.RawMessage is []byte)
	Time			time.Time		`json:"-"`		// proto send time
}

func (p *Proto) Reset() {
	*p = emptyProto
}


func (p *Proto) ReadTCP(rr *bufio.Reader) (err error) {
	var (
		bodyLen int
		packLen int32
		buf     []byte
	)
	if buf, err = rr.Pop(RawHeaderSize); err != nil {
		return
	}
	packLen = binary.BigEndian.Int32(buf[PackOffset:HeaderOffset])
	if packLen > MaxPackSize {
		return ErrProtoPackLen
	}
	
	p.HeaderLen = binary.BigEndian.Int16(buf[HeaderOffset:VerOffset])
	if p.HeaderLen != RawHeaderSize {
		return ErrProtoHeaderLen
	}
	if bodyLen = int(packLen - int32(p.HeaderLen)); bodyLen > 0 {
		p.Body, err = rr.Pop(bodyLen)
	} else {
		p.Body = nil
	}
	
	p.Ver = binary.BigEndian.Int16(buf[VerOffset : OperationOffset])
	p.OpCode = binary.BigEndian.Int32(buf[OperationOffset : MsgIdOffset])
	p.MsgId = binary.BigEndian.Int64(buf[MsgIdOffset : EnsureOffset])

	return
}

func (p *Proto) WriteTCP(wr *bufio.Writer) (err error) {
	var (
		buf     []byte
		packLen int32
	)
	if p.OpCode == define.OP_RAW_MSG {
		_, err = wr.Write(p.Body)
		return
	}
	packLen = RawHeaderSize + int32(len(p.Body))
	p.HeaderLen = RawHeaderSize
	if buf, err = wr.Peek(RawHeaderSize); err != nil {
		return
	}
	binary.BigEndian.PutInt32(buf[PackOffset:], packLen)
	binary.BigEndian.PutInt16(buf[HeaderOffset:], p.HeaderLen)
	binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
	binary.BigEndian.PutInt32(buf[OperationOffset:], p.OpCode)
	binary.BigEndian.PutInt64(buf[MsgIdOffset:], p.MsgId)
	
	if p.Ensure {
		buf[EnsureOffset] = 1
	} else {
		buf[EnsureOffset] = 0
	}
	//binary.BigEndian.PutInt32(buf[ExpireOffset:], p.Expire)

	if p.Body != nil {
		_, err = wr.Write(p.Body)
	}
	return
}

func (p *Proto) ReadWebsocket(wr *websocket.Conn) (err error) {
	err = websocket.JSON.Receive(wr, p)
	return
}

func (p *Proto) WriteWebsocket(wr *websocket.Conn) (err error) {
	if p.Body == nil {
		p.Body = emptyJSONBody
	}
	
	err = websocket.JSON.Send(wr, p)
	return
}
