package output

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// applyJQ applies a JQ filter expression to JSON bytes and returns the result as a string.
func applyJQ(data []byte, filter string) (string, error) {
	query, err := gojq.Parse(filter)
	if err != nil {
		return "", fmt.Errorf("invalid JQ expression %q: %w", filter, err)
	}

	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		return "", fmt.Errorf("JSON parse for JQ: %w", err)
	}

	iter := query.Run(input)
	var buf bytes.Buffer
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return "", fmt.Errorf("JQ error: %w", err)
		}
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", err
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}
	return string(bytes.TrimRight(buf.Bytes(), "\n")), nil
}
