package notify

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/appleboy/gorush/logx"
)

// extractHeaders converts a slice of "Key: Value" strings into a header map.
//
// Each entry is split on the first colon only, so values that themselves
// contain colons — bearer tokens, timestamped signatures (e.g. "t=1700:v1=ab"),
// or URLs with ports — are preserved intact instead of being truncated or
// silently dropped. Keys and values are trimmed; entries without a colon or
// with an empty key are skipped.
func extractHeaders(headers []string) map[string]string {
	result := make(map[string]string, len(headers))
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(parts[1])
	}
	return result
}

// DispatchFeedback sends a feedback log entry to a specified URL via an HTTP POST request.
//
// Parameters:
//   - ctx: The context for the HTTP request.
//   - log: The log entry to be sent as feedback.
//   - url: The destination URL for the feedback.
//   - timeout: The timeout duration for the HTTP request in seconds.
//   - header: A slice of strings representing additional headers to be included in the request.
//
// Returns:
//   - error: An error if the request fails or the response status is not OK.
func DispatchFeedback(
	ctx context.Context,
	log logx.LogPushEntry,
	url string,
	timeout int64,
	header []string,
) error {
	if url == "" {
		return errors.New("url can't be empty")
	}

	payload, err := json.Marshal(log)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	for k, v := range extractHeaders(header) {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// The per-request timeout is enforced via the request context above. We
	// deliberately do NOT mutate the shared feedbackClient.Timeout here: that
	// field is package-global, so writing it on every call races with other
	// in-flight feedback dispatches. http.Client.Do is safe for concurrent use
	// as long as the client itself is not mutated.
	resp, err := feedbackClient.Do(req)

	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to send feedback")
	}

	return nil
}
