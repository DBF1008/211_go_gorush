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

// buildCLIRequest builds and normalizes a PushNotification from CLI options.
// It maps Token → To and Topic → Topic, then calls CheckMessage to fold To
// into Tokens and perform platform-specific validation. To is cleared after
// normalization so that downstream PushToXyz functions (which may call
// CheckMessage again internally) do not duplicate the token.
func buildCLIRequest(opts CLISendOptions, platform int) (*notify.PushNotification, error) {
	req := &notify.PushNotification{
		Platform: platform,
		Message:  opts.Message,
		Title:    opts.Title,
		To:       opts.Token,
		Topic:    opts.Topic,
	}

	// Normalize: fold To → Tokens and validate.
	if err := notify.CheckMessage(req); err != nil {
		return nil, err
	}

	// Clear To so a second CheckMessage (inside PushToAndroid/PushToHuawei)
	// does not re-append it and cause duplicate sends.
	req.To = ""

	return req, nil
}

// SendAndroidNotification sends an Android notification via CLI.
func SendAndroidNotification(ctx context.Context, cfg *config.ConfYaml, opts CLISendOptions) error {
	cfg.Android.Enabled = true

	req, err := buildCLIRequest(opts, core.PlatFormAndroid)
	if err != nil {
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

	req, err := buildCLIRequest(opts, core.PlatFormHuawei)
	if err != nil {
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

	req, err := buildCLIRequest(opts, core.PlatFormIos)
	if err != nil {
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
