package messages

import (
	"bufio"
	"errors"
	"strings"
)

const (
	rateMsgVersion   = 0x1
	rateMsgDelimiter = '\n'
)

type RateMsg struct {
	MsgID string
	Code  string
}

func (msg *RateMsg) ID() string {
	return msg.MsgID
}

//nolint:errcheck
func (msg *RateMsg) Serialize(br *bufio.ReadWriter) error {
	br.WriteByte(rateMsgVersion)
	br.WriteString(msg.MsgID)
	br.WriteByte(rateMsgDelimiter)
	br.WriteString(msg.Code)

	return br.WriteByte(rateMsgDelimiter)
}

func (msg *RateMsg) Deserialize(br *bufio.Reader) error {
	var (
		err     error
		version byte
	)

	if version, err = br.ReadByte(); err != nil {
		return err
	}

	if version != rateMsgVersion {
		return errors.New("invalid version")
	}

	if msg.MsgID, err = br.ReadString(rateMsgDelimiter); err != nil {
		return err
	}
	msg.MsgID = strings.TrimSuffix(msg.MsgID, string(rateMsgDelimiter))

	if msg.Code, err = br.ReadString(rateMsgDelimiter); err != nil {
		return err
	}
	msg.Code = strings.TrimSuffix(msg.Code, string(rateMsgDelimiter))

	return nil
}
