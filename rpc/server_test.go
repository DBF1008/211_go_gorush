package rpc

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/core"
	"github.com/appleboy/gorush/logx"
	"github.com/appleboy/gorush/notify"
	"github.com/appleboy/gorush/rpc/proto"
	"github.com/appleboy/gorush/status"

	"github.com/golang-queue/queue"
	qcore "github.com/golang-queue/queue/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestMain(m *testing.M) {
	cfg, _ := config.LoadConf()
	cfg.Core.Mode = "test"
	if err := status.InitAppStatus(cfg); err != nil {
		log.Fatal(err)
	}
	code := m.Run()
	os.Exit(code)
}

func initTestConfig() *config.ConfYaml {
	cfg, _ := config.LoadConf()
	cfg.Core.Mode = "test"
	return cfg
}

func newTestQueue(cfg *config.ConfYaml, fn func(ctx context.Context, msg qcore.TaskMessage) error) *queue.Queue {
	if fn == nil {
		fn = func(ctx context.Context, msg qcore.TaskMessage) error {
			return nil
		}
	}
	return queue.NewPool(
		cfg.Core.WorkerNum,
		queue.WithFn(fn),
		queue.WithLogger(logx.QueueLogger()),
	)
}

func TestSafeIntToInt32(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int32
		wantErr bool
	}{
		{"Valid int32", 123, 123, false},
		{"Max int32", math.MaxInt32, math.MaxInt32, false},
		{"Min int32", math.MinInt32, math.MinInt32, false},
		{"Overflow int32", math.MaxInt32 + 1, 0, true},
		{"Underflow int32", math.MinInt32 - 1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeIntToInt32(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("safeIntToInt32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("safeIntToInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendValidationErrors(t *testing.T) {
	cfg := initTestConfig()
	cfg.Android.Enabled = true

	q := newTestQueue(cfg, nil)
	defer q.Release()

	srv := NewServer(cfg, q)

	t.Run("empty tokens and no topic returns InvalidArgument", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "hello",
			Tokens:   []string{},
		}
		resp, err := srv.Send(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		st, ok := grpcstatus.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "at least one device token")
	})

	t.Run("empty string token returns InvalidArgument for iOS", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormIos,
			Message:  "hello",
			Tokens:   []string{""},
		}
		resp, err := srv.Send(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		st, ok := grpcstatus.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "device token cannot be empty")
	})

	t.Run("over 500 tokens for Android returns InvalidArgument", func(t *testing.T) {
		tokens := make([]string, 501)
		for i := range tokens {
			tokens[i] = "token"
		}
		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "hello",
			Tokens:   tokens,
		}
		resp, err := srv.Send(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		st, ok := grpcstatus.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "500")
	})
}

func TestSendPlatformDisabled(t *testing.T) {
	cfg := initTestConfig()
	// Disable all platforms
	cfg.Ios.Enabled = false
	cfg.Android.Enabled = false
	cfg.Huawei.Enabled = false

	q := newTestQueue(cfg, nil)
	defer q.Release()

	srv := NewServer(cfg, q)

	t.Run("Android notification when disabled returns zero counts", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "hello",
			Tokens:   []string{"token1", "token2"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(0), resp.Counts)
	})

	t.Run("iOS notification when disabled returns zero counts", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormIos,
			Message:  "hello",
			Tokens:   []string{"token1"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(0), resp.Counts)
	})

	t.Run("Huawei notification when disabled returns zero counts", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormHuawei,
			Message:  "hello",
			Tokens:   []string{"token1"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(0), resp.Counts)
	})
}

func TestSendValidNotificationAsync(t *testing.T) {
	cfg := initTestConfig()
	cfg.Android.Enabled = true
	cfg.Ios.Enabled = true
	cfg.Core.Sync = false // async mode

	q := newTestQueue(cfg, func(ctx context.Context, msg qcore.TaskMessage) error {
		return nil
	})
	defer q.Release()

	srv := NewServer(cfg, q)

	t.Run("Android notification with tokens counts correctly", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "hello android",
			Tokens:   []string{"token1", "token2", "token3"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(3), resp.Counts)
	})

	t.Run("iOS notification with single token", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormIos,
			Message:  "hello ios",
			Tokens:   []string{"abcdef1234567890"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(1), resp.Counts)
	})
}

func TestSendTopicNotification(t *testing.T) {
	cfg := initTestConfig()
	cfg.Android.Enabled = true
	cfg.Core.Sync = false

	q := newTestQueue(cfg, func(ctx context.Context, msg qcore.TaskMessage) error {
		return nil
	})
	defer q.Release()

	srv := NewServer(cfg, q)

	t.Run("topic-only notification counts as 1", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "topic message",
			Topic:    "/topics/news",
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(1), resp.Counts)
	})

	t.Run("tokens plus topic counts both", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "mixed message",
			Tokens:   []string{"token1", "token2"},
			Topic:    "/topics/news",
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		// 2 tokens + 1 topic = 3
		assert.Equal(t, int32(3), resp.Counts)
	})
}

func TestSendSyncMode(t *testing.T) {
	cfg := initTestConfig()
	cfg.Android.Enabled = true
	cfg.Core.Sync = true
	cfg.Queue.Engine = "local" // local queue enables sync mode

	t.Run("sync mode with successful queue dispatch", func(t *testing.T) {
		q := newTestQueue(cfg, func(ctx context.Context, msg qcore.TaskMessage) error {
			return nil
		})
		defer q.Release()

		srv := NewServer(cfg, q)

		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "sync message",
			Tokens:   []string{"token1", "token2"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(2), resp.Counts)
	})

	t.Run("sync mode disabled platform returns zero counts", func(t *testing.T) {
		cfgDisabled := initTestConfig()
		cfgDisabled.Android.Enabled = false
		cfgDisabled.Core.Sync = true
		cfgDisabled.Queue.Engine = "local"

		q := newTestQueue(cfgDisabled, nil)
		defer q.Release()

		srv := NewServer(cfgDisabled, q)

		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "should be filtered",
			Tokens:   []string{"token1"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(0), resp.Counts)
	})
}

func TestSendNotificationMapping(t *testing.T) {
	cfg := initTestConfig()
	cfg.Android.Enabled = true
	cfg.Core.Sync = false // async mode so queue fn is called by worker

	captured := make(chan []byte, 1)
	q := newTestQueue(cfg, func(ctx context.Context, msg qcore.TaskMessage) error {
		// In async mode, Queue() serializes the message to bytes.
		// Capture the raw payload for later deserialization.
		if b := msg.Payload(); len(b) > 0 {
			captured <- b
		}
		return nil
	})
	defer q.Release()

	srv := NewServer(cfg, q)

	t.Run("proto fields map correctly to PushNotification", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{
			"key1": "value1",
			"key2": float64(42),
		})
		require.NoError(t, err)

		req := &proto.NotificationRequest{
			ID:               "test-id-123",
			Platform:         core.PlatFormAndroid,
			Message:          "test message",
			Title:            "Test Title",
			Topic:            "/topics/test",
			Tokens:           []string{"token1"},
			Badge:            5,
			Category:         "MESSAGE",
			Sound:            "default",
			ContentAvailable: true,
			ThreadID:         "thread-1",
			MutableContent:   true,
			Image:            "https://example.com/img.png",
			Priority:         proto.NotificationRequest_HIGH,
			PushType:         "alert",
			Development:      true,
			Alert: &proto.Alert{
				Title:    "Alert Title",
				Body:     "Alert Body",
				Subtitle: "Alert Subtitle",
			},
			Data: data,
			FcmOptions: &proto.FCMOptions{
				AnalyticsLabel: "test-label",
			},
		}

		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		// 1 token + 1 topic = 2
		assert.Equal(t, int32(2), resp.Counts)

		// Wait for async worker to process
		var payload []byte
		select {
		case payload = <-captured:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for notification to be processed")
		}

		// Deserialize the captured notification
		var got notify.PushNotification
		err = json.Unmarshal(payload, &got)
		require.NoError(t, err)

		// Verify mapping
		assert.Equal(t, "test-id-123", got.ID)
		assert.Equal(t, core.PlatFormAndroid, got.Platform)
		assert.Equal(t, "test message", got.Message)
		assert.Equal(t, "Test Title", got.Title)
		assert.Equal(t, "/topics/test", got.Topic)
		assert.Equal(t, []string{"token1"}, got.Tokens)
		assert.Equal(t, 5, *got.Badge)
		assert.Equal(t, "MESSAGE", got.Category)
		assert.Equal(t, "default", got.Sound)
		assert.True(t, got.ContentAvailable)
		assert.Equal(t, "thread-1", got.ThreadID)
		assert.True(t, got.MutableContent)
		assert.Equal(t, "https://example.com/img.png", got.Image)
		assert.Equal(t, "high", got.Priority)
		assert.Equal(t, "alert", got.PushType)
		assert.True(t, got.Development)
		assert.Equal(t, "Alert Title", got.Alert.Title)
		assert.Equal(t, "Alert Body", got.Alert.Body)
		assert.Equal(t, "Alert Subtitle", got.Alert.Subtitle)
		assert.Equal(t, "value1", got.Data["key1"])
		require.NotNil(t, got.FCMOptions)
		assert.Equal(t, "test-label", got.FCMOptions.AnalyticsLabel)
	})
}

func TestSendMixedPlatformFiltering(t *testing.T) {
	cfg := initTestConfig()
	cfg.Android.Enabled = true
	cfg.Ios.Enabled = false
	cfg.Core.Sync = false

	q := newTestQueue(cfg, func(ctx context.Context, msg qcore.TaskMessage) error {
		return nil
	})
	defer q.Release()

	srv := NewServer(cfg, q)

	t.Run("disabled platform is filtered but valid platform passes", func(t *testing.T) {
		// Send an iOS notification when iOS is disabled
		req := &proto.NotificationRequest{
			Platform: core.PlatFormIos,
			Message:  "ios message",
			Tokens:   []string{"token1"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(0), resp.Counts) // filtered out
	})

	t.Run("enabled platform is not filtered", func(t *testing.T) {
		req := &proto.NotificationRequest{
			Platform: core.PlatFormAndroid,
			Message:  "android message",
			Tokens:   []string{"token1", "token2"},
		}
		resp, err := srv.Send(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(2), resp.Counts)
	})
}
