package app

import (
	"context"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/core"
	"github.com/appleboy/gorush/logx"
	"github.com/appleboy/gorush/notify"
	"github.com/appleboy/gorush/status"
)

// CLISendOptions contains options for sending notifications via CLI.
type CLISendOptions struct {
	Token   string
	Message string
	Title   string
	Topic   string
}

// newCLIPushNotification builds a PushNotification from CLI send options with
// the same field mapping for every platform: a single --token always lands in
// Tokens, and --topic always lands in Topic. Platform-specific interpretation of
// Topic (FCM/HMS topic routing vs. the APNs topic header) is left to IsTopic and
// the per-platform push functions, keeping the CLI construction consistent.
//
// Tokens (not To) is used deliberately: CheckMessage folds To into Tokens on
// every call, and PushToAndroid/PushToHuawei call CheckMessage again internally,
// so populating To here would append the token twice.
func newCLIPushNotification(platform int, opts CLISendOptions) *notify.PushNotification {
	req := &notify.PushNotification{
		Platform: platform,
		Message:  opts.Message,
		Title:    opts.Title,
	}

	if opts.Token != "" {
		req.Tokens = []string{opts.Token}
	}

	if opts.Topic != "" {
		req.Topic = opts.Topic
	}

	return req
}

// SendAndroidNotification sends an Android notification via CLI.
func SendAndroidNotification(ctx context.Context, cfg *config.ConfYaml, opts CLISendOptions) error {
	cfg.Android.Enabled = true

	req := newCLIPushNotification(core.PlatFormAndroid, opts)

	if err := notify.CheckMessage(req); err != nil {
		return err
	}

	if err := status.InitAppStatus(cfg); err != nil {
		return err
	}

	if _, err := notify.PushToAndroid(ctx, req, cfg); err != nil {
		return err
	}

	return nil
}

// SendHuaweiNotification sends a Huawei notification via CLI.
func SendHuaweiNotification(ctx context.Context, cfg *config.ConfYaml, opts CLISendOptions) error {
	cfg.Huawei.Enabled = true

	req := newCLIPushNotification(core.PlatFormHuawei, opts)

	if err := notify.CheckMessage(req); err != nil {
		return err
	}

	if err := status.InitAppStatus(cfg); err != nil {
		return err
	}

	if _, err := notify.PushToHuawei(ctx, req, cfg); err != nil {
		return err
	}

	return nil
}

// SendIOSNotification sends an iOS notification via CLI.
func SendIOSNotification(ctx context.Context, cfg *config.ConfYaml, opts CLISendOptions) error {
	cfg.Ios.Enabled = true

	req := newCLIPushNotification(core.PlatFormIos, opts)

	if err := notify.CheckMessage(req); err != nil {
		return err
	}

	if err := status.InitAppStatus(cfg); err != nil {
		return err
	}

	if err := notify.InitAPNSClient(ctx, cfg); err != nil {
		return err
	}

	if _, err := notify.PushToIOS(ctx, req, cfg); err != nil {
		return err
	}

	return nil
}

// SendNotification sends a notification based on platform type.
func SendNotification(
	ctx context.Context,
	platform int,
	cfg *config.ConfYaml,
	opts CLISendOptions,
) error {
	switch platform {
	case core.PlatFormAndroid:
		return SendAndroidNotification(ctx, cfg, opts)
	case core.PlatFormHuawei:
		return SendHuaweiNotification(ctx, cfg, opts)
	case core.PlatFormIos:
		return SendIOSNotification(ctx, cfg, opts)
	default:
		logx.LogError.Fatalf("unsupported platform: %d", platform)
		return nil
	}
}
