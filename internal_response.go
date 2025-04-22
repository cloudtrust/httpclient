package httpclient

import (
	"encoding/json"

	"gopkg.in/h2non/gentleman.v2"
)

// Internal Response is the only way to let ReadContent be testable
type internalResponse struct {
	gentlemanResponse *gentleman.Response
	bytes             []byte
}

func buildInternalResponse(resp *gentleman.Response) *internalResponse {
	return &internalResponse{
		gentlemanResponse: resp,
		bytes:             nil,
	}
}

func (ir *internalResponse) StatusCode() int {
	return ir.gentlemanResponse.StatusCode
}

func (ir *internalResponse) GetHeader(name string) string {
	return ir.gentlemanResponse.Header.Get(name)
}

func (ir *internalResponse) Bytes() []byte {
	if ir.bytes == nil {
		ir.bytes = ir.gentlemanResponse.Bytes()
	}
	return ir.bytes
}

func (ir *internalResponse) JSON(data any) error {
	return json.Unmarshal(ir.Bytes(), data)
}

func (ir *internalResponse) String() string {
	return string(ir.Bytes())
}
