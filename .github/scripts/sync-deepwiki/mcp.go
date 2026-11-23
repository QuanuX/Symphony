package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const deepWikiEndpoint = "https://mcp.deepwiki.com/mcp"

// DeepWiki returns a single JSON-RPC response as JSON or a server-sent event.
func decodeWikiResponse(data []byte) (string, error) {
	var response struct {
		ID     int                       `json:"id"`
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError bool
			Content []struct{ Type, Text string }
		}
	}
	data = bytes.TrimSpace(data)
	if !bytes.HasPrefix(data, []byte("{")) {
		// SSE can split one event over several data fields or precede it with a heartbeat.
		stream := strings.ReplaceAll(string(data), "\r\n", "\n")
		data = nil
		for _, event := range strings.Split(stream, "\n\n") {
			var fields []string
			for line := range strings.SplitSeq(event, "\n") {
				if field, ok := strings.CutPrefix(line, "data:"); ok {
					fields = append(fields, strings.TrimPrefix(field, " "))
				}
			}
			candidate := []byte(strings.Join(fields, "\n"))
			var envelope struct{ ID int }
			if json.Unmarshal(candidate, &envelope) == nil && envelope.ID == 1 {
				data = candidate
				break
			}
		}
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("invalid DeepWiki response: %w", err)
	}
	if response.Error != nil {
		return "", fmt.Errorf("DeepWiki RPC error: %s", response.Error.Message)
	}
	if response.ID != 1 || response.Result.IsError {
		return "", errors.New("DeepWiki did not return a successful tool result")
	}
	var parts []string
	for _, block := range response.Result.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return "", errors.New("DeepWiki returned no text")
	}
	return strings.Join(parts, "\n"), nil
}

func wikiText(tool, inputDir string) (string, error) {
	var filename string
	switch tool {
	case "read_wiki_structure":
		filename = "structure.sse"
	case "read_wiki_contents":
		filename = "contents.sse"
	default:
		return "", fmt.Errorf("unknown DeepWiki tool %q", tool)
	}
	if inputDir != "" {
		data, err := os.ReadFile(filepath.Join(inputDir, filename))
		if err != nil {
			return "", err
		}
		return decodeWikiResponse(data)
	}

	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": map[string]string{"repoName": "QuanuX/Symphony"}},
	})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest(http.MethodPost, deepWikiEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	client := &http.Client{Timeout: 90 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepWiki %s returned %s", tool, response.Status)
	}
	const limit = 32 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return "", err
	}
	if len(data) > limit {
		return "", fmt.Errorf("DeepWiki %s exceeds %d bytes", tool, limit)
	}
	return decodeWikiResponse(data)
}
