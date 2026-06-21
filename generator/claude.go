package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type choice struct {
	Message message `json:"message"`
}

type response struct {
	Choices []choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// retriableError wraps an HTTP-level error and signals whether retry is appropriate.
type retriableError struct {
	statusCode int
	msg        string
}

func (e *retriableError) Error() string { return e.msg }

func isRetriable(err error) bool {
	var re *retriableError
	if err == nil {
		return false
	}
	// type-assert manually since errors.As needs a pointer-to-pointer
	if re, ok := err.(*retriableError); ok {
		return re.statusCode == 429 || re.statusCode >= 500
	}
	_ = re
	return false
}

// Generate calls the OpenRouter API and returns the CLAUDE.md content.
// timeoutSec controls the per-attempt HTTP timeout; retries on 429 / 5xx (up to 3 attempts).
func Generate(apiKey, model, prompt string, timeoutSec int) (string, error) {
	body := request{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(2<<uint(attempt-1)) * time.Second // 2s, 4s
			time.Sleep(backoff)
		}

		content, err := doRequest(client, apiKey, data)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !isRetriable(err) {
			break
		}
	}
	return "", lastErr
}

func doRequest(client *http.Client, apiKey string, data []byte) (string, error) {
	req, err := http.NewRequest("POST", openRouterURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/elev1e1n/spelunk-md")
	req.Header.Set("X-Title", "spelunk-md")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return "", &retriableError{
			statusCode: resp.StatusCode,
			msg:        fmt.Sprintf("http %d: %s", resp.StatusCode, snippet),
		}
	}

	var result response
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse error: %w\nraw: %s", err, string(raw))
	}

	if result.Error != nil {
		return "", fmt.Errorf("api error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}

// WriteFile writes content to the given path.
func WriteFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	return nil
}
