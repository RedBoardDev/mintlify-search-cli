package render

import (
	"encoding/json"
	"fmt"
	"io"
)

type jsonRenderer struct{}

// Render emits a minified JSON encoding of payload followed by a newline.
// Field order follows the struct declaration, which Go's encoding/json
// guarantees is stable.
func (r *jsonRenderer) Render(w io.Writer, payload any) error {
	if payload == nil {
		return fmt.Errorf("render: nil payload")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("render json: %w", err)
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}
