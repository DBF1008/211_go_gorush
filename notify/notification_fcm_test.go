package notify

import (
	"context"
	"os"
	"reflect"
	"testing"

	"firebase.google.com/go/v4/messaging"
	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissingAndroidCredential(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = ""

	err := CheckPushConf(cfg)

	require.Error(t, err)
	assert.Equal(t, "missing fcm credential data", err.Error())
}

func TestMissingKeyForInitFCMClient(t *testing.T) {
	cfg, _ := config.LoadConf()
	cfg.Android.Credential = ""
	cfg.Android.KeyPath = ""
	client, err := InitFCMClient(context.Background(), cfg)

	assert.Nil(t, client)
	require.Error(t, err)
	assert.Equal(t, "missing fcm credential data", err.Error())
}

func TestPushToAndroidWrongToken(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	req := &PushNotification{
		Tokens:   []string{"aaaaaa", "bbbbb"},
		Platform: core.PlatFormAndroid,
		Message:  "Welcome",
	}

	// Android Success count: 0, Failure count: 2
	resp, err := PushToAndroid(context.Background(), req, cfg)
	require.NoError(t, err)
	assert.Len(t, resp.Logs, 2)
}

func TestPushToAndroidRightTokenForJSONLog(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")
	// log for json
	cfg.Log.Format = "json"

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := &PushNotification{
		Tokens:   []string{androidToken},
		Platform: core.PlatFormAndroid,
		Message:  "Welcome",
	}

	resp, err := PushToAndroid(context.Background(), req, cfg)
	require.NoError(t, err)
	assert.Empty(t, resp.Logs)
}

func TestPushToAndroidRightTokenForStringLog(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := &PushNotification{
		Tokens:   []string{androidToken},
		Platform: core.PlatFormAndroid,
		Message:  "Welcome",
	}

	resp, err := PushToAndroid(context.Background(), req, cfg)
	require.NoError(t, err)
	assert.Empty(t, resp.Logs)
}

func TestFCMMessage(t *testing.T) {
	var err error

	// the message must specify at least one registration ID
	req := &PushNotification{
		Message: "Test",
		Tokens:  []string{},
	}

	err = CheckMessage(req)
	require.Error(t, err)

	// ignore check token length if send topic message
	req = &PushNotification{
		Message:  "Test",
		Platform: core.PlatFormAndroid,
		Topic:    "/topics/foo-bar",
	}

	err = CheckMessage(req)
	require.NoError(t, err)

	// "condition": "'dogs' in topics || 'cats' in topics",
	req = &PushNotification{
		Message:   "Test",
		Platform:  core.PlatFormAndroid,
		Condition: "'dogs' in topics || 'cats' in topics",
	}

	err = CheckMessage(req)
	require.NoError(t, err)

	// the message may specify at most 1000 registration IDs
	req = &PushNotification{
		Message:  "Test",
		Platform: core.PlatFormAndroid,
		Tokens:   make([]string, 501),
	}

	err = CheckMessage(req)
	require.Error(t, err)

	// Pass
	req = &PushNotification{
		Message:  "Test",
		Platform: core.PlatFormAndroid,
		Tokens:   []string{"XXXXXXXXX"},
	}

	err = CheckMessage(req)
	require.NoError(t, err)
}

func TestAndroidNotificationStructure(t *testing.T) {
	test := "test"
	req := &PushNotification{
		Tokens:         []string{"a", "b"},
		Message:        "Welcome",
		To:             test,
		Priority:       HIGH,
		MutableContent: true,
		Title:          test,
		Sound:          test,
		Data: D{
			"a": "1",
			"b": 2,
			"json": map[string]any{
				"c": "3",
				"d": 4,
			},
		},
		Notification: &messaging.Notification{
			Title: test,
			Body:  "",
		},
	}

	messages := GetAndroidNotification(req)

	assert.Equal(t, test, messages[0].Notification.Title)
	assert.Equal(t, "Welcome", messages[0].Notification.Body)
	assert.Equal(t, "1", messages[0].Data["a"])
	assert.Equal(t, "2", messages[0].Data["b"])
	assert.JSONEq(t, `{"c":"3","d":4}`, messages[0].Data["json"])
	assert.NotNil(t, messages[0].APNS)
	assert.Equal(t, req.Sound, messages[0].APNS.Payload.Aps.Sound)
	assert.Equal(t, req.MutableContent, messages[0].APNS.Payload.Aps.MutableContent)

	// test empty body
	req = &PushNotification{
		Tokens: []string{"a", "b"},
		To:     test,
		Notification: &messaging.Notification{
			Body: "",
		},
	}
	messages = GetAndroidNotification(req)

	assert.Empty(t, messages[0].Notification.Body)
}

func TestAndroidBackgroundNotificationStructure(t *testing.T) {
	data := map[string]any{
		"a": "1",
		"b": 2,
		"json": map[string]any{
			"c": "3",
			"d": 4,
		},
	}
	req := &PushNotification{
		Tokens:           []string{"a", "b"},
		Priority:         HIGH,
		ContentAvailable: true,
		Data:             data,
	}

	messages := GetAndroidNotification(req)

	assert.Equal(t, "1", messages[0].Data["a"])
	assert.Equal(t, "2", messages[0].Data["b"])
	assert.JSONEq(t, `{"c":"3","d":4}`, messages[0].Data["json"])
	assert.NotNil(t, messages[0].APNS)
	assert.Equal(t, req.ContentAvailable, messages[0].APNS.Payload.Aps.ContentAvailable)
	assert.True(t, reflect.DeepEqual(data, messages[0].APNS.Payload.Aps.CustomData))
}

// Tests for refactored helper functions

func TestSetupFCMNotification(t *testing.T) {
	tests := []struct {
		name        string
		req         *PushNotification
		wantTitle   string
		wantBody    string
		wantImage   string
		wantMutable bool
	}{
		{
			name: "title message and image",
			req: &PushNotification{
				Tokens:  []string{"token"},
				Title:   "Test Title",
				Message: "Test Message",
				Image:   "https://example.com/image.png",
			},
			wantTitle: "Test Title",
			wantBody:  "Test Message",
			wantImage: "https://example.com/image.png",
		},
		{
			name: "mutable content",
			req: &PushNotification{
				Tokens:         []string{"token"},
				Title:          "Title",
				MutableContent: true,
			},
			wantTitle:   "Title",
			wantMutable: true,
		},
		{
			name: "empty notification fields",
			req: &PushNotification{
				Tokens: []string{"token"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := GetAndroidNotification(tt.req)
			if len(messages) > 0 {
				msg := messages[0]
				if tt.wantTitle != "" {
					assert.Equal(t, tt.wantTitle, msg.Notification.Title)
				}
				if tt.wantBody != "" {
					assert.Equal(t, tt.wantBody, msg.Notification.Body)
				}
				if tt.wantImage != "" {
					assert.Equal(t, tt.wantImage, msg.Notification.ImageURL)
				}
				if tt.wantMutable {
					assert.NotNil(t, msg.APNS)
					assert.True(t, msg.APNS.Payload.Aps.MutableContent)
				}
			}
		})
	}
}

func TestSetupFCMContentAvailable(t *testing.T) {
	data := D{"key": "value"}
	req := &PushNotification{
		Tokens:           []string{"token"},
		ContentAvailable: true,
		Data:             data,
	}

	messages := GetAndroidNotification(req)
	assert.Len(t, messages, 1)
	msg := messages[0]

	assert.NotNil(t, msg.APNS)
	assert.Equal(t, "5", msg.APNS.Headers["apns-priority"])
	assert.True(t, msg.APNS.Payload.Aps.ContentAvailable)
	assert.Equal(t, data, D(msg.APNS.Payload.Aps.CustomData))
}

func TestSetAPNSSound(t *testing.T) {
	tests := []struct {
		name      string
		req       *PushNotification
		wantSound string
	}{
		{
			name: "sound with no existing APNS",
			req: &PushNotification{
				Tokens: []string{"token"},
				Sound:  "default",
			},
			wantSound: "default",
		},
		{
			name: "sound with existing APNS from mutable content",
			req: &PushNotification{
				Tokens:         []string{"token"},
				Title:          "Title",
				MutableContent: true,
				Sound:          "custom.aiff",
			},
			wantSound: "custom.aiff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := GetAndroidNotification(tt.req)
			if len(messages) > 0 {
				msg := messages[0]
				assert.NotNil(t, msg.APNS)
				assert.Equal(t, tt.wantSound, msg.APNS.Payload.Aps.Sound)
			}
		})
	}
}

func TestSetupFCMSound(t *testing.T) {
	req := &PushNotification{
		Tokens:   []string{"token"},
		Sound:    "test.aiff",
		Priority: HIGH,
	}

	messages := GetAndroidNotification(req)
	assert.Len(t, messages, 1)
	msg := messages[0]

	// Check APNS sound is set
	assert.NotNil(t, msg.APNS)
	assert.Equal(t, "test.aiff", msg.APNS.Payload.Aps.Sound)

	// Check Android config is set with sound
	assert.NotNil(t, msg.Android)
	assert.Equal(t, "test.aiff", msg.Android.Notification.Sound)
	assert.Equal(t, HIGH, msg.Android.Priority)
}

func TestConvertDataToStringMap(t *testing.T) {
	tests := []struct {
		name string
		data D
		want map[string]string
	}{
		{
			name: "empty data",
			data: D{},
			want: nil,
		},
		{
			name: "string values",
			data: D{"key1": "value1", "key2": "value2"},
			want: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name: "mixed values",
			data: D{"str": "text", "num": 42, "bool": true},
			want: map[string]string{"str": "text", "num": "42", "bool": "true"},
		},
		{
			name: "nested object",
			data: D{"nested": map[string]any{"a": 1}},
			want: map[string]string{"nested": `{"a":1}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDataToStringMap(tt.data)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuildFCMMessage(t *testing.T) {
	req := &PushNotification{
		Tokens: []string{"token"},
		Title:  "Title",
		Notification: &messaging.Notification{
			Title: "Notification Title",
			Body:  "Notification Body",
		},
		Android: &messaging.AndroidConfig{
			Priority: HIGH,
		},
		Webpush: &messaging.WebpushConfig{
			Headers: map[string]string{"TTL": "3600"},
		},
		FCMOptions: &messaging.FCMOptions{
			AnalyticsLabel: "test-label",
		},
	}

	data := map[string]string{"key": "value"}
	msg := buildFCMMessage(req, data)

	assert.Equal(t, req.Notification, msg.Notification)
	assert.Equal(t, req.Android, msg.Android)
	assert.Equal(t, req.Webpush, msg.Webpush)
	assert.Equal(t, req.FCMOptions, msg.FCMOptions)
	assert.Equal(t, data, msg.Data)
}

func TestGetAndroidNotificationWithTopic(t *testing.T) {
	req := &PushNotification{
		Platform:  core.PlatFormAndroid,
		Topic:     "test-topic",
		Condition: "'dogs' in topics",
		Message:   "Topic message",
	}

	messages := GetAndroidNotification(req)
	assert.Len(t, messages, 1)
	assert.Equal(t, "test-topic", messages[0].Topic)
	assert.Equal(t, "'dogs' in topics", messages[0].Condition)
}

func TestGetAndroidNotificationWithTokens(t *testing.T) {
	req := &PushNotification{
		Tokens:  []string{"token1", "token2", "token3"},
		Message: "Multi token message",
	}

	messages := GetAndroidNotification(req)
	assert.Len(t, messages, 3)
	assert.Equal(t, "token1", messages[0].Token)
	assert.Equal(t, "token2", messages[1].Token)
	assert.Equal(t, "token3", messages[2].Token)
}

func TestGetAndroidNotificationWithTopicAndTokens(t *testing.T) {
	req := &PushNotification{
		Platform: core.PlatFormAndroid,
		Topic:    "test-topic",
		Tokens:   []string{"token1", "token2"},
		Message:  "Combined message",
	}

	messages := GetAndroidNotification(req)
	// 1 for topic + 2 for tokens = 3 messages
	assert.Len(t, messages, 3)
	assert.Equal(t, "test-topic", messages[0].Topic)
	assert.Equal(t, "token1", messages[1].Token)
	assert.Equal(t, "token2", messages[2].Token)
}

// Composable tests: verify that multiple FCM features stack correctly
// instead of overwriting each other.

func TestComposableContentAvailableAndMutableContent(t *testing.T) {
	req := &PushNotification{
		Tokens:           []string{"token"},
		Title:            "Title",
		Message:          "Body",
		ContentAvailable: true,
		MutableContent:   true,
		Data:             D{"key": "value"},
	}

	messages := GetAndroidNotification(req)
	require.Len(t, messages, 1)
	msg := messages[0]

	// Both APNS flags must co-exist (previously content_available clobbered mutable_content).
	require.NotNil(t, msg.APNS)
	require.NotNil(t, msg.APNS.Payload)
	require.NotNil(t, msg.APNS.Payload.Aps)
	assert.True(t, msg.APNS.Payload.Aps.MutableContent,
		"mutable_content must survive content_available")
	assert.True(t, msg.APNS.Payload.Aps.ContentAvailable,
		"content_available must be set")
	assert.Equal(t, "5", msg.APNS.Headers["apns-priority"])
	assert.Equal(t, D{"key": "value"}, D(msg.APNS.Payload.Aps.CustomData))

	// Notification body must still be present.
	require.NotNil(t, msg.Notification)
	assert.Equal(t, "Title", msg.Notification.Title)
	assert.Equal(t, "Body", msg.Notification.Body)
}

func TestComposableMutableContentSoundAndContentAvailable(t *testing.T) {
	req := &PushNotification{
		Tokens:           []string{"token"},
		Title:            "Title",
		Message:          "Body",
		Image:            "https://example.com/image.png",
		MutableContent:   true,
		ContentAvailable: true,
		Sound:            "chime.aiff",
		Priority:         HIGH,
		Data:             D{"extra": "data"},
	}

	messages := GetAndroidNotification(req)
	require.Len(t, messages, 1)
	msg := messages[0]

	// --- APNS: every capability must be present ---
	require.NotNil(t, msg.APNS)
	aps := msg.APNS.Payload.Aps
	assert.True(t, aps.MutableContent, "mutable_content must survive")
	assert.True(t, aps.ContentAvailable, "content_available must survive")
	assert.Equal(t, "chime.aiff", aps.Sound, "sound must survive")
	assert.Equal(t, "5", msg.APNS.Headers["apns-priority"])
	assert.Equal(t, D{"extra": "data"}, D(aps.CustomData))

	// --- Top-level notification ---
	require.NotNil(t, msg.Notification)
	assert.Equal(t, "Title", msg.Notification.Title)
	assert.Equal(t, "Body", msg.Notification.Body)
	assert.Equal(t, "https://example.com/image.png", msg.Notification.ImageURL)

	// --- Android: sound and priority must be propagated ---
	require.NotNil(t, msg.Android)
	assert.Equal(t, HIGH, msg.Android.Priority)
	require.NotNil(t, msg.Android.Notification)
	assert.Equal(t, "chime.aiff", msg.Android.Notification.Sound)
}

func TestComposablePreservesCallerAPNSHeaders(t *testing.T) {
	req := &PushNotification{
		Tokens:           []string{"token"},
		ContentAvailable: true,
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-push-type": "background",
				"apns-topic":     "com.example.app",
			},
		},
	}

	messages := GetAndroidNotification(req)
	require.Len(t, messages, 1)
	msg := messages[0]

	require.NotNil(t, msg.APNS)
	// Caller-supplied headers must be preserved.
	assert.Equal(t, "background", msg.APNS.Headers["apns-push-type"])
	assert.Equal(t, "com.example.app", msg.APNS.Headers["apns-topic"])
	// content-available header must be merged in.
	assert.Equal(t, "5", msg.APNS.Headers["apns-priority"])
}

func TestComposableSoundAugmentsExistingAndroidConfig(t *testing.T) {
	req := &PushNotification{
		Tokens:   []string{"token"},
		Sound:    "alert.caf",
		Priority: HIGH,
		// Caller pre-sets an Android config with a custom TTL.
		Android: &messaging.AndroidConfig{
			TTL: "3600s",
		},
	}

	messages := GetAndroidNotification(req)
	require.Len(t, messages, 1)
	msg := messages[0]

	require.NotNil(t, msg.Android)
	// Caller's TTL must be preserved.
	assert.Equal(t, "3600s", msg.Android.TTL)
	// Priority should be filled in from req.Priority.
	assert.Equal(t, HIGH, msg.Android.Priority)
	// Sound must be applied.
	require.NotNil(t, msg.Android.Notification)
	assert.Equal(t, "alert.caf", msg.Android.Notification.Sound)
}

func TestComposableSoundDoesNotOverrideExistingAndroidPriority(t *testing.T) {
	req := &PushNotification{
		Tokens:   []string{"token"},
		Sound:    "alert.caf",
		Priority: HIGH,
		// Caller explicitly sets a different priority on the Android config.
		Android: &messaging.AndroidConfig{
			Priority: "normal",
		},
	}

	messages := GetAndroidNotification(req)
	require.Len(t, messages, 1)
	msg := messages[0]

	require.NotNil(t, msg.Android)
	// Caller's explicit priority must NOT be overwritten.
	assert.Equal(t, "normal", msg.Android.Priority)
	assert.Equal(t, "alert.caf", msg.Android.Notification.Sound)
}
