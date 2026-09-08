package messages

type RateMsg struct {
	MsgID string
	Code  string
}

func (msg *RateMsg) ID() string {
	return msg.MsgID
}
