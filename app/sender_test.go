package app

import (
	"testing"

	"github.com/appleboy/gorush/core"

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

func TestBuildCLIRequest_TokenOnly(t *testing.T) {
	platforms := []struct {
		name     string
		platform int
	}{
		{"iOS", core.PlatFormIos},
		{"Android", core.PlatFormAndroid},
		{"Huawei", core.PlatFormHuawei},
	}

	for _, p := range platforms {
		t.Run(p.name, func(t *testing.T) {
			opts := CLISendOptions{
				Token:   "device-token-abc",
				Message: "hello",
				Title:   "greeting",
			}

			req, err := buildCLIRequest(opts, p.platform)
			require.NoError(t, err)

			assert.Equal(t, p.platform, req.Platform)
			assert.Equal(t, "hello", req.Message)
			assert.Equal(t, "greeting", req.Title)
			assert.Equal(t, []string{"device-token-abc"}, req.Tokens)
			assert.Empty(t, req.Topic)
			// To must be cleared to prevent downstream CheckMessage duplication
			assert.Empty(t, req.To)
		})
	}
}

func TestBuildCLIRequest_TopicOnly_Android(t *testing.T) {
	opts := CLISendOptions{
		Topic:   "news",
		Message: "breaking news",
	}

	req, err := buildCLIRequest(opts, core.PlatFormAndroid)
	require.NoError(t, err)

	assert.Equal(t, core.PlatFormAndroid, req.Platform)
	assert.Equal(t, "news", req.Topic)
	assert.Empty(t, req.Tokens)
	assert.Empty(t, req.To)
	assert.True(t, req.IsTopic())
}

func TestBuildCLIRequest_TopicOnly_Huawei(t *testing.T) {
	opts := CLISendOptions{
		Topic:   "huawei-topic",
		Message: "huawei msg",
	}

	req, err := buildCLIRequest(opts, core.PlatFormHuawei)
	require.NoError(t, err)

	assert.Equal(t, core.PlatFormHuawei, req.Platform)
	assert.Equal(t, "huawei-topic", req.Topic)
	assert.Empty(t, req.Tokens)
	assert.Empty(t, req.To)
	assert.True(t, req.IsTopic())
}

func TestBuildCLIRequest_TopicOnly_IOS(t *testing.T) {
	// iOS uses Topic as the APNs bundle ID, not as a fan-out topic.
	// IsTopic() returns false for iOS since it's not a pub/sub topic.
	opts := CLISendOptions{
		Topic: "com.example.app",
		Token: "device-token",
	}

	req, err := buildCLIRequest(opts, core.PlatFormIos)
	require.NoError(t, err)

	assert.Equal(t, core.PlatFormIos, req.Platform)
	assert.Equal(t, "com.example.app", req.Topic)
	assert.Equal(t, []string{"device-token"}, req.Tokens)
	assert.False(t, req.IsTopic())
}

func TestBuildCLIRequest_TokenAndTopic_Android(t *testing.T) {
	opts := CLISendOptions{
		Token:   "device-token",
		Topic:   "news",
		Message: "msg",
	}

	req, err := buildCLIRequest(opts, core.PlatFormAndroid)
	require.NoError(t, err)

	assert.Equal(t, core.PlatFormAndroid, req.Platform)
	assert.Equal(t, "news", req.Topic)
	assert.Equal(t, []string{"device-token"}, req.Tokens)
	assert.Empty(t, req.To)
	assert.True(t, req.IsTopic())
}

func TestBuildCLIRequest_TokenAndTopic_Huawei(t *testing.T) {
	opts := CLISendOptions{
		Token:   "huawei-token",
		Topic:   "huawei-topic",
		Message: "msg",
	}

	req, err := buildCLIRequest(opts, core.PlatFormHuawei)
	require.NoError(t, err)

	assert.Equal(t, core.PlatFormHuawei, req.Platform)
	assert.Equal(t, "huawei-topic", req.Topic)
	assert.Equal(t, []string{"huawei-token"}, req.Tokens)
	assert.Empty(t, req.To)
	assert.True(t, req.IsTopic())
}

func TestBuildCLIRequest_NoTokenNoTopic_Android(t *testing.T) {
	opts := CLISendOptions{
		Message: "msg",
	}

	_, err := buildCLIRequest(opts, core.PlatFormAndroid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "please provide at least one device token")
}

func TestBuildCLIRequest_NoTokenNoTopic_Huawei(t *testing.T) {
	opts := CLISendOptions{
		Message: "msg",
	}

	_, err := buildCLIRequest(opts, core.PlatFormHuawei)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "please provide at least one device token")
}

func TestBuildCLIRequest_NoTokenNoTopic_IOS(t *testing.T) {
	opts := CLISendOptions{
		Message: "msg",
	}

	_, err := buildCLIRequest(opts, core.PlatFormIos)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "please provide at least one device token")
}

func TestBuildCLIRequest_EmptyTokenString_IOS(t *testing.T) {
	opts := CLISendOptions{
		Token: "",
	}

	_, err := buildCLIRequest(opts, core.PlatFormIos)
	require.Error(t, err)
	// Empty token string with no topic → "please provide at least one device token"
	assert.Contains(t, err.Error(), "please provide at least one device token")
}

func TestBuildCLIRequest_HuaweiTopicGoesToTopicField(t *testing.T) {
	// This is the key regression test for the Huawei topic bug:
	// previously --topic was written to req.To instead of req.Topic,
	// causing IsTopic() to return false and the topic string to be
	// treated as a device token.
	opts := CLISendOptions{
		Topic:   "my-huawei-topic",
		Message: "msg",
	}

	req, err := buildCLIRequest(opts, core.PlatFormHuawei)
	require.NoError(t, err)

	// Topic must be in the Topic field, NOT in To or Tokens
	assert.Equal(t, "my-huawei-topic", req.Topic)
	assert.Empty(t, req.To)
	assert.Empty(t, req.Tokens)
	assert.True(t, req.IsTopic())
}

func TestBuildCLIRequest_AndroidTokenUsesTokensNotTo(t *testing.T) {
	// Regression test: previously Android put the token into req.To
	// instead of letting CheckMessage normalize it into req.Tokens.
	opts := CLISendOptions{
		Token:   "android-device-token",
		Message: "msg",
	}

	req, err := buildCLIRequest(opts, core.PlatFormAndroid)
	require.NoError(t, err)

	assert.Equal(t, []string{"android-device-token"}, req.Tokens)
	assert.Empty(t, req.To)
}

func TestBuildCLIRequest_SemanticConsistencyAcrossPlatforms(t *testing.T) {
	// Verify that the same CLISendOptions produce semantically consistent
	// requests across all three platforms (modulo platform-specific fields).
	opts := CLISendOptions{
		Token:   "shared-token",
		Topic:   "shared-topic",
		Message: "shared-message",
		Title:   "shared-title",
	}

	platforms := []int{core.PlatFormIos, core.PlatFormAndroid, core.PlatFormHuawei}
	requests := make(map[int]*struct {
		tokens []string
		topic  string
		to     string
	})

	for _, p := range platforms {
		req, err := buildCLIRequest(opts, p)
		require.NoError(t, err)

		requests[p] = &struct {
			tokens []string
			topic  string
			to     string
		}{
			tokens: req.Tokens,
			topic:  req.Topic,
			to:     req.To,
		}
	}

	// All platforms: token normalized into Tokens, To cleared
	for _, p := range platforms {
		r := requests[p]
		assert.Equal(t, []string{"shared-token"}, r.tokens, "platform %d: Tokens", p)
		assert.Equal(t, "shared-topic", r.topic, "platform %d: Topic", p)
		assert.Empty(t, r.to, "platform %d: To should be cleared", p)
	}
}
