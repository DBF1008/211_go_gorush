package rpc

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/core"
	"github.com/appleboy/gorush/notify"
	"github.com/appleboy/gorush/rpc/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// fakeSender records whether it was invoked and returns a configurable result,
// standing in for notify.SendNotification so Send's result mapping can be tested
// without contacting real push services.
type fakeSender struct {
	called bool
	resp   *notify.ResponsePush
	err    error
}

func (f *fakeSender) send(
	_ context.Context,
	_ *notify.PushNotification,
	_ *config.ConfYaml,
) (*notify.ResponsePush, error) {
	f.called = true
	return f.resp, f.err
}

func newTestServer(t *testing.T) (*Server, *fakeSender) {
	t.Helper()
	cfg, err := config.LoadConf()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	s := NewServer(cfg)
	f := &fakeSender{resp: &notify.ResponsePush{}}
	s.send = f.send
	return s, f
}

func TestServerSend_InvalidRequest(t *testing.T) {
	s, f := newTestServer(t)
	s.cfg.Android.Enabled = true

	// Android with neither tokens nor topic fails CheckMessage.
	_, err := s.Send(context.Background(), &proto.NotificationRequest{
		Platform: core.PlatFormAndroid,
		Message:  "test",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v (err=%v)", status.Code(err), err)
	}
	if f.called {
		t.Fatal("send should not be called for an invalid request")
	}
}

func TestServerSend_PlatformDisabled(t *testing.T) {
	s, f := newTestServer(t)
	s.cfg.Android.Enabled = false

	_, err := s.Send(context.Background(), &proto.NotificationRequest{
		Platform: core.PlatFormAndroid,
		Tokens:   []string{"token"},
		Message:  "test",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v (err=%v)", status.Code(err), err)
	}
	if f.called {
		t.Fatal("send should not be called when the platform is disabled")
	}
}

func TestServerSend_TopicOnlySuccess(t *testing.T) {
	s, f := newTestServer(t)
	s.cfg.Android.Enabled = true

	// Topic-only push (no tokens) is valid for Android and counts the topic.
	reply, err := s.Send(context.Background(), &proto.NotificationRequest{
		Platform: core.PlatFormAndroid,
		Topic:    "/topics/news",
		Message:  "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.called {
		t.Fatal("send should be called")
	}
	if !reply.Success {
		t.Fatal("expected success=true")
	}
	if reply.Counts != 1 {
		t.Fatalf("expected counts=1 for a topic-only push, got %d", reply.Counts)
	}
}

func TestServerSend_Success(t *testing.T) {
	s, f := newTestServer(t)
	s.cfg.Android.Enabled = true

	reply, err := s.Send(context.Background(), &proto.NotificationRequest{
		Platform: core.PlatFormAndroid,
		Tokens:   []string{"a", "b"},
		Message:  "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.called {
		t.Fatal("send should be called")
	}
	if !reply.Success || reply.Counts != 2 {
		t.Fatalf("expected success=true counts=2, got success=%v counts=%d", reply.Success, reply.Counts)
	}
}

func TestServerSend_SendFailure(t *testing.T) {
	s, f := newTestServer(t)
	s.cfg.Android.Enabled = true
	f.err = errors.New("boom")

	_, err := s.Send(context.Background(), &proto.NotificationRequest{
		Platform: core.PlatFormAndroid,
		Tokens:   []string{"token"},
		Message:  "test",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v (err=%v)", status.Code(err), err)
	}
	if !f.called {
		t.Fatal("send should be called before the failure is reported")
	}
}

// const gRPCAddr = "localhost:9000"

// func initTest() *config.ConfYaml {
// 	cfg, _ := config.LoadConf()
// 	cfg.Core.Mode = "test"
// 	return cfg
// }

// func TestGracefulShutDownGRPCServer(t *testing.T) {
// 	cfg := initTest()
// 	cfg.GRPC.Enabled = true
// 	cfg.GRPC.Port = "9000"
// 	cfg.Log.Format = "json"

// 	// Run gRPC server
// 	ctx, gRPCContextCancel := context.WithCancel(context.Background())
// 	go func() {
// 		if err := RunGRPCServer(ctx, cfg); err != nil {
// 			panic(err)
// 		}
// 	}()

// 	// gRPC client conn
// 	conn, err := grpc.Dial(
// 		gRPCAddr,
// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
// 		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
// 	) // wait for server ready
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	// Stop gRPC server
// 	go gRPCContextCancel()

// 	// wait for client connection would be closed
// 	for conn.GetState() != connectivity.TransientFailure {
// 	}
// 	conn.Close()
// }
