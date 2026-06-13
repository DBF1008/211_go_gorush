package notify

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/logx"

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
		name  string
		input []string
		want  map[string]string
	}{
		{
			name:  "simple value with space after colon",
			input: []string{"x-gorush-token: 1234"},
			want:  map[string]string{"x-gorush-token": "1234"},
		},
		{
			name:  "simple value without space",
			input: []string{"x-api-key:1234567890"},
			want:  map[string]string{"x-api-key": "1234567890"},
		},
		{
			name:  "bearer token value containing a colon",
			input: []string{"Authorization: Bearer header.payload:signature"},
			want:  map[string]string{"Authorization": "Bearer header.payload:signature"},
		},
		{
			name:  "timestamped signature value",
			input: []string{"X-Signature: t=1700000000:v1=deadbeef"},
			want:  map[string]string{"X-Signature": "t=1700000000:v1=deadbeef"},
		},
		{
			name:  "url value with port",
			input: []string{"X-Callback-Url: https://example.com:8080/hook"},
			want:  map[string]string{"X-Callback-Url": "https://example.com:8080/hook"},
		},
		{
			name:  "multiple headers including colon values",
			input: []string{"x-api-key: secret", "X-Signature: t=1:v1=ab:cd"},
			want: map[string]string{
				"x-api-key":   "secret",
				"X-Signature": "t=1:v1=ab:cd",
			},
		},
		{
			name:  "entry without a colon is skipped",
			input: []string{"not-a-header"},
			want:  map[string]string{},
		},
		{
			name:  "entry with empty key is skipped",
			input: []string{": orphan-value"},
			want:  map[string]string{},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, extractHeaders(tt.input))
		})
	}
}

// TestFeedbackHeaderWithColonValue verifies that header values containing
// colons (bearer tokens, timestamped signatures) reach the destination intact.
// The mock server rejects the request unless both values arrive unchanged, so a
// regression that truncates or drops them fails this test.
func TestFeedbackHeaderWithColonValue(t *testing.T) {
	const (
		sigHeader  = "t=1700000000:v1=deadbeefcafe"
		authHeader = "Bearer header.payload:signature"
	)

	httpMock := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Signature") != sigHeader ||
				r.Header.Get("Authorization") != authHeader {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer httpMock.Close()

	err := DispatchFeedback(
		context.Background(),
		logx.LogPushEntry{},
		httpMock.URL,
		5,
		[]string{
			"X-Signature: " + sigHeader,
			"Authorization: " + authHeader,
		},
	)
	require.NoError(t, err)
}

// TestConcurrentDispatchFeedback exercises many simultaneous feedback
// dispatches against a single endpoint. Run with -race it guards against the
// previous data race where DispatchFeedback mutated the shared client's
// Timeout on every call. It also asserts every request arrives with the
// correct header, proving header handling is not cross-contaminated.
func TestConcurrentDispatchFeedback(t *testing.T) {
	const (
		concurrency = 64
		tokenValue  = "1234"
	)

	var hits atomic.Int32
	httpMock := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			if r.Header.Get("x-gorush-token") != tokenValue {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer httpMock.Close()

	errs := make([]error, concurrency)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := range concurrency {
		go func(idx int) {
			defer wg.Done()
			errs[idx] = DispatchFeedback(
				context.Background(),
				logx.LogPushEntry{},
				httpMock.URL,
				5,
				[]string{"x-gorush-token: " + tokenValue},
			)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "dispatch %d failed", i)
	}
	require.Equal(t, int32(concurrency), hits.Load())
}
