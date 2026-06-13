package notify

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/logx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyFeedbackURL(t *testing.T) {
	cfg, _ := config.LoadConf()
	logEntry := logx.LogPushEntry{
		ID:       "",
		Type:     "",
		Platform: "",
		Token:    "",
		Message:  "",
		Error:    "",
	}

	err := DispatchFeedback(
		context.Background(),
		logEntry,
		cfg.Core.FeedbackURL,
		cfg.Core.FeedbackTimeout,
		cfg.Core.FeedbackHeader,
	)
	require.Error(t, err)
}

func TestHTTPErrorInFeedbackCall(t *testing.T) {
	cfg, _ := config.LoadConf()
	cfg.Core.FeedbackURL = "http://test.example.com/api/"
	logEntry := logx.LogPushEntry{
		ID:       "",
		Type:     "",
		Platform: "",
		Token:    "",
		Message:  "",
		Error:    "",
	}

	err := DispatchFeedback(
		context.Background(),
		logEntry,
		cfg.Core.FeedbackURL,
		cfg.Core.FeedbackTimeout,
		cfg.Core.FeedbackHeader,
	)
	require.Error(t, err)
}

func TestSuccessfulFeedbackCall(t *testing.T) {
	// Mock http server
	httpMock := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dispatch" {
				// check http header
				if r.Header.Get("x-gorush-token") != "1234" {
					panic("x-gorush-token header is not set")
				}

				w.Header().Add("Content-Type", "application/json")
				_, err := w.Write([]byte(`{}`))
				if err != nil {
					log.Println(err)
					panic(err)
				}
			}
		}),
	)
	defer httpMock.Close()

	cfg, _ := config.LoadConf()
	cfg.Core.FeedbackURL = httpMock.URL + "/dispatch"
	cfg.Core.FeedbackHeader = []string{
		"x-gorush-token: 1234",
	}
	logEntry := logx.LogPushEntry{
		ID:       "",
		Type:     "",
		Platform: "",
		Token:    "",
		Message:  "",
		Error:    "",
	}

	err := DispatchFeedback(
		context.Background(),
		logEntry,
		cfg.Core.FeedbackURL,
		cfg.Core.FeedbackTimeout,
		cfg.Core.FeedbackHeader,
	)
	require.NoError(t, err)
}

func TestExtractHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:  "simple header",
			input: []string{"Content-Type: application/json"},
			expected: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:  "header without space after colon",
			input: []string{"X-Token:abc123"},
			expected: map[string]string{
				"X-Token": "abc123",
			},
		},
		{
			name:  "Bearer token with colons in value",
			input: []string{"Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload:signature"},
			expected: map[string]string{
				"Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload:signature",
			},
		},
		{
			name:  "HMAC signature with colons",
			input: []string{"X-Signature: sha256=abc123:def456:ghi789"},
			expected: map[string]string{
				"X-Signature": "sha256=abc123:def456:ghi789",
			},
		},
		{
			name:  "timestamp signature with colons",
			input: []string{"X-Timestamp-Sig: ts=1234567890:sig=abcdef:nonce=xyz"},
			expected: map[string]string{
				"X-Timestamp-Sig": "ts=1234567890:sig=abcdef:nonce=xyz",
			},
		},
		{
			name:  "multiple headers mixed",
			input: []string{"X-Token: 1234", "Authorization: Bearer abc:def:ghi", "X-Request-ID: req-001"},
			expected: map[string]string{
				"X-Token":        "1234",
				"Authorization":  "Bearer abc:def:ghi",
				"X-Request-ID":   "req-001",
			},
		},
		{
			name:     "header without colon is skipped",
			input:    []string{"InvalidHeader", "X-Good: value"},
			expected: map[string]string{"X-Good": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHeaders(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFeedbackHeadersWithColonsInValue(t *testing.T) {
	// Integration test: verify that headers containing colons in values
	// are transmitted correctly to the feedback endpoint.
	var receivedAuth, receivedSig string

	httpMock := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			receivedSig = r.Header.Get("X-Signature")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}),
	)
	defer httpMock.Close()

	cfg, _ := config.LoadConf()
	logEntry := logx.LogPushEntry{
		ID:       "test-id",
		Type:     "success",
		Platform: "ios",
		Token:    "token123",
		Message:  "hello",
		Error:    "",
	}

	bearerToken := "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0:signature-part"
	hmacSig := "sha256=aabbccdd:eeff0011:22334455"

	err := DispatchFeedback(
		context.Background(),
		logEntry,
		httpMock.URL,
		cfg.Core.FeedbackTimeout,
		[]string{
			"Authorization: " + bearerToken,
			"X-Signature: " + hmacSig,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, bearerToken, receivedAuth)
	assert.Equal(t, hmacSig, receivedSig)
}

func TestConcurrentDispatchFeedback(t *testing.T) {
	// Verify that concurrent DispatchFeedback calls do not race on shared state.
	// Run with -race to detect data races.
	requestCount := 0
	var mu sync.Mutex

	httpMock := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify each request carries its own header
			token := r.Header.Get("X-Req-Token")
			assert.NotEmpty(t, token)

			mu.Lock()
			requestCount++
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}),
	)
	defer httpMock.Close()

	const numRequests = 50

	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			logEntry := logx.LogPushEntry{
				ID:       "concurrent-test",
				Type:     "success",
				Platform: "ios",
				Token:    "token",
				Message:  "msg",
				Error:    "",
			}
			// Use different timeouts to stress-test that we no longer
			// mutate a shared http.Client.Timeout.
			var timeout int64 = 5
			if idx%2 == 0 {
				timeout = 10
			}

			err := DispatchFeedback(
				context.Background(),
				logEntry,
				httpMock.URL,
				timeout,
				[]string{"X-Req-Token: token-value"},
			)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	mu.Lock()
	assert.Equal(t, numRequests, requestCount)
	mu.Unlock()
}

func TestDispatchFeedbackNonOKStatus(t *testing.T) {
	httpMock := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer httpMock.Close()

	logEntry := logx.LogPushEntry{
		ID:       "test",
		Type:     "fail",
		Platform: "android",
		Token:    "tok",
		Message:  "msg",
		Error:    "some error",
	}

	err := DispatchFeedback(
		context.Background(),
		logEntry,
		httpMock.URL,
		10,
		nil,
	)
	require.Error(t, err)
	assert.Equal(t, "failed to send feedback", err.Error())
}
