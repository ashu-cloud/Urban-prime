package locationv1

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

type jsonProtoCodec struct{}

func (jsonProtoCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (jsonProtoCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (jsonProtoCodec) Name() string                       { return "proto" }

func init() {
	encoding.RegisterCodec(jsonProtoCodec{})
}
