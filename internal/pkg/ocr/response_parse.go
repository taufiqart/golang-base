package ocr

import (
	"encoding/json"
	"fmt"
	"strings"
)

func completionContent(raw []byte) (string, error) {
	var out chatCompletionResponse
	if err := json.Unmarshal(raw, &out); err == nil && len(out.Choices) > 0 && out.Choices[0].Message.Content != "" {
		return out.Choices[0].Message.Content, nil
	}
	var b strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk); err != nil {
			return "", fmt.Errorf("decode ocr stream: %w", err)
		}
		if len(chunk.Choices) > 0 {
			b.WriteString(chunk.Choices[0].Delta.Content)
		}
	}
	if b.Len() == 0 {
		return "", ErrFailed
	}
	return b.String(), nil
}

func firstJSONObject(content string) (string, error) {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return "", ErrFailed
	}
	return content[start : end+1], nil
}
