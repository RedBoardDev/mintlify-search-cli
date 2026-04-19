package render

import (
	"encoding/json"
	"fmt"
	"io"
)

type rawRenderer struct{}

// Render accepts a RawPayload and emits its json.RawMessage verbatim with a
// trailing newline. For any other payload type, it falls back to standard
// JSON marshaling (useful when the caller hasn't wrapped in RawPayload yet).
func (r *rawRenderer) Render(w io.Writer, payload any) error {
	var data []byte
	switch p := payload.(type) {
	case RawPayload:
		data = append([]byte(nil), p.Result...)
	case *RawPayload:
		data = append([]byte(nil), p.Result...)
	case json.RawMessage:
		data = append([]byte(nil), p...)
	case nil:
		return fmt.Errorf("render raw: nil payload")
	default:
		var err error
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("render raw: %w", err)
		}
	}
	if len(data) == 0 {
		return fmt.Errorf("render raw: empty payload")
	}
	if data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	_, err := w.Write(data)
	return err
}
