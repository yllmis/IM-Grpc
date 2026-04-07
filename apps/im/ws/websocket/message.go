package websocket

type FrameType uint8

const (
	FrameData FrameType = 0x0
	FramePing FrameType = 0x1
	FrameErr  FrameType = 0x9 // 返回给前端
	// FrameHeaders      FrameType = 0x1
	// FramePriority     FrameType = 0x2
	// FrameRSTStream    FrameType = 0x3
	// FrameSettings     FrameType = 0x4
	// FramePushPromise  FrameType = 0x5

	// FrameGoAway       FrameType = 0x7
	// FrameWindowUpdate FrameType = 0x8
	// FrameContinuation FrameType = 0x9
)

type Message struct {
	FrameType `json:"frameType"`
	Method    string      `json:"method"`
	FormId    string      `json:"formId"`
	Data      interface{} // map[string]interface{}
}

func NewMessage(formId string, data interface{}) *Message {
	return &Message{
		FrameType: FrameData,
		FormId:    formId,
		Data:      data,
	}
}

func NewErrMessgae(err error) *Message {
	return &Message{
		FrameType: FrameErr,
		Data:      err.Error(),
	}
}
