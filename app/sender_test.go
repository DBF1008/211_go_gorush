package app

import (
	"context"
	"testing"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/core"
	"github.com/appleboy/gorush/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLISendOptions(t *testing.T) {
	opts := CLISendOptions{
		Token:   "test-token",
		Message: "test-message",
		Title:   "test-title",
		Topic:   "test-topic",
	}

	assert.Equal(t, "test-token", opts.Token)
	assert.Equal(t, "test-message", opts.Message)
	assert.Equal(t, "test-title", opts.Title)
	assert.Equal(t, "test-topic", opts.Topic)
}

func TestSendNotification_UnsupportedPlatform(t *testing.T) {
	// Test that unsupported platform is handled
	// Note: This would call logx.LogError.Fatalf in production,
	// so we can't easily test it without mocking.
	// This test mainly documents the expected behavior.
	assert.Equal(t, 1, core.PlatFormIos)
	assert.Equal(t, 2, core.PlatFormAndroid)
	assert.Equal(t, 3, core.PlatFormHuawei)
}

// TestNewCLIPushNotification verifies that every platform maps the CLI --token
// into Tokens and --topic into Topic, never into the legacy To field. This is the
// cross-platform consistent construction the senders rely on.
func TestNewCLIPushNotification(t *testing.T) {
	platforms := []struct {
		name     string
		platform int
	}{
		{"ios", core.PlatFormIos},
		{"android", core.PlatFormAndroid},
		{"huawei", core.PlatFormHuawei},
	}

	for _, p := range platforms {
		t.Run(p.name, func(t *testing.T) {
			req := newCLIPushNotification(p.platform, CLISendOptions{
				Token:   "device-token",
				Message: "hello",
				Title:   "title",
				Topic:   "my-topic",
			})

			assert.Equal(t, p.platform, req.Platform)
			assert.Equal(t, "hello", req.Message)
			assert.Equal(t, "title", req.Title)
			// Token always lands in Tokens, identically for every platform.
			assert.Equal(t, []string{"device-token"}, req.Tokens)
			// Topic always lands in Topic, never in To.
			assert.Equal(t, "my-topic", req.Topic)
			assert.Empty(t, req.To)
		})
	}
}

// TestNewCLIPushNotification_EmptyFields verifies that absent --token / --topic
// leave the corresponding fields unset rather than producing empty entries.
func TestNewCLIPushNotification_EmptyFields(t *testing.T) {
	req := newCLIPushNotification(core.PlatFormAndroid, CLISendOptions{
		Message: "only message",
	})

	assert.Empty(t, req.Tokens)
	assert.Empty(t, req.Topic)
	assert.Empty(t, req.To)
	assert.Equal(t, "only message", req.Message)
}

// TestSenderHuaweiTopicRoutesAsTopic guards the bug fix: a Huawei --topic must be
// recognised as a real topic (IsTopic == true) so it is dispatched via the topic
// path, instead of being shoved into To and sent as a device token.
func TestSenderHuaweiTopicRoutesAsTopic(t *testing.T) {
	req := newCLIPushNotification(core.PlatFormHuawei, CLISendOptions{
		Topic:   "weather-news",
		Message: "storm incoming",
	})

	// The topic must be in Topic, not To, and must not be treated as a token.
	assert.Equal(t, "weather-news", req.Topic)
	assert.Empty(t, req.To)
	assert.Empty(t, req.Tokens)

	// A topic-only Huawei request is valid (no device token required) ...
	require.NoError(t, notify.CheckMessage(req))
	// ... and is routed through the topic path.
	assert.True(t, req.IsTopic())
	// CheckMessage must not have synthesised a token from the topic.
	assert.Empty(t, req.Tokens)
}

// TestSenderTopicSemanticsByPlatform documents and locks in the per-platform
// meaning of --topic after construction + validation:
//   - Android/Huawei: a topic is a routing target (IsTopic true, no token needed).
//   - iOS: Topic is the APNs topic header, not a route, so a device token is still
//     required and IsTopic stays false.
func TestSenderTopicSemanticsByPlatform(t *testing.T) {
	t.Run("android topic only is a topic", func(t *testing.T) {
		req := newCLIPushNotification(core.PlatFormAndroid, CLISendOptions{Topic: "t"})
		require.NoError(t, notify.CheckMessage(req))
		assert.True(t, req.IsTopic())
	})

	t.Run("huawei topic only is a topic", func(t *testing.T) {
		req := newCLIPushNotification(core.PlatFormHuawei, CLISendOptions{Topic: "t"})
		require.NoError(t, notify.CheckMessage(req))
		assert.True(t, req.IsTopic())
	})

	t.Run("ios topic only still needs a token", func(t *testing.T) {
		req := newCLIPushNotification(core.PlatFormIos, CLISendOptions{Topic: "com.example.app"})
		assert.False(t, req.IsTopic())
		err := notify.CheckMessage(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "please provide at least one device token")
	})
}

// TestSenderValidationConsistency verifies the converged validation chain: with no
// token and no topic, all three senders fail fast with the same error before any
// status/client initialisation or network call. Previously Android skipped
// CheckMessage and behaved differently from iOS/Huawei.
func TestSenderValidationConsistency(t *testing.T) {
	senders := []struct {
		name string
		send func(context.Context, *config.ConfYaml, CLISendOptions) error
	}{
		{"ios", SendIOSNotification},
		{"android", SendAndroidNotification},
		{"huawei", SendHuaweiNotification},
	}

	for _, s := range senders {
		t.Run(s.name, func(t *testing.T) {
			cfg, err := config.LoadConf()
			require.NoError(t, err)

			err = s.send(context.Background(), cfg, CLISendOptions{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "please provide at least one device token")
		})
	}
}

// TestSenderTokenNotDuplicated guards the deliberate use of Tokens (not To) in the
// shared builder: PushToAndroid/PushToHuawei call CheckMessage internally, so the
// validation chain may run more than once. Running it repeatedly must not grow the
// token list, which would happen if the token were stored in To.
func TestSenderTokenNotDuplicated(t *testing.T) {
	req := newCLIPushNotification(core.PlatFormAndroid, CLISendOptions{Token: "tok"})

	require.NoError(t, notify.CheckMessage(req))
	require.NoError(t, notify.CheckMessage(req))

	assert.Equal(t, []string{"tok"}, req.Tokens)
}
