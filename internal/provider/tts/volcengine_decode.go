package tts

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func decodeBase64Audio(data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("decode audio: %w", err)
	}
	return b, nil
}
