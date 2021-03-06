package codecs

import (
	"bytes"

	"github.com/ewe-studios/sabuhp"

	"github.com/influx6/npkg/nerror"
	"github.com/vmihailenco/msgpack/v5"
)

var _ sabuhp.Codec = (*MessageMsgPackCodec)(nil)

type MessageMsgPackCodec struct{}

func (j *MessageMsgPackCodec) Encode(message sabuhp.Message) ([]byte, error) {
	message.Parts = nil
	var buf bytes.Buffer
	if encodedErr := msgpack.NewEncoder(&buf).Encode(message); encodedErr != nil {
		return nil, nerror.WrapOnly(encodedErr)
	}
	return buf.Bytes(), nil
}

func (j *MessageMsgPackCodec) Decode(b []byte) (sabuhp.Message, error) {
	var message sabuhp.Message
	if jsonErr := msgpack.NewDecoder(bytes.NewBuffer(b)).Decode(&message); jsonErr != nil {
		return message, nerror.WrapOnly(jsonErr)
	}
	message.Future = nil
	return message, nil
}
