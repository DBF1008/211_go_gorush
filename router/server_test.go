package router

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/core"
	"github.com/appleboy/gorush/logx"
	"github.com/appleboy/gorush/notify"
	"github.com/appleboy/gorush/status"

	"github.com/appleboy/gofight/v2"
	"github.com/buger/jsonparser"
	"github.com/gin-gonic/gin"
	"github.com/golang-queue/queue"
	qcore "github.com/golang-queue/queue/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	goVersion   = runtime.Version()
	q           *queue.Queue
	testKeyPath = "../certificate/certificate-valid.pem"
)

func TestMain(m *testing.M) {
	cfg := initTest()
	if err := status.InitAppStatus(cfg); err != nil {
		log.Fatal(err)
	}

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	if _, err := notify.InitFCMClient(context.Background(), cfg); err != nil {
		log.Fatal(err)
	}

	q = queue.NewPool(
		cfg.Core.WorkerNum,
		queue.WithFn(func(ctx context.Context, msg qcore.TaskMessage) error {
			_, err := notify.SendNotification(ctx, msg, cfg)
			return err
		}),
		queue.WithLogger(logx.QueueLogger()),
	)

	code := m.Run()
	defer func() {
		q.Release()
		os.Exit(code)
	}()
}

func initTest() *config.ConfYaml {
	cfg, _ := config.LoadConf()
	cfg.Core.Mode = "test"
	return cfg
}

// testRequest is testing url string if server is running
func testRequest(t *testing.T, url string) {
	tr := &http.Transport{
		//nolint:gosec // InsecureSkipVerify is needed for testing with self-signed certificates
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: tr,
	}
	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		url,
		nil,
	)
	resp, err := client.Do(req)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("close body err:", err)
		}
	}()

	require.NoError(t, err)

	_, ioerr := io.ReadAll(resp.Body)
	require.NoError(t, ioerr)
	assert.Equal(t, "200 OK", resp.Status, "should get a 200")
}

func TestPrintGoRushVersion(t *testing.T) {
	SetVersion("3.0.0")
	SetCommit("abcdefg")
	ver := GetVersion()
	PrintGoRushVersion()

	assert.Equal(t, "3.0.0", ver)
}

func TestRunNormalServer(t *testing.T) {
	cfg := initTest()

	gin.SetMode(gin.TestMode)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, RunHTTPServer(ctx, cfg, q))
	}()

	defer func() {
		// close the server
		cancel()
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	testRequest(t, "http://localhost:8088/api/stat/go")
}

func TestRunTLSServer(t *testing.T) {
	cfg := initTest()

	cfg.Core.SSL = true
	cfg.Core.Port = "8087"
	cfg.Core.CertPath = "../certificate/localhost.cert"
	cfg.Core.KeyPath = "../certificate/localhost.key"

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, RunHTTPServer(ctx, cfg, q))
	}()

	defer func() {
		// close the server
		cancel()
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	testRequest(t, "https://localhost:8087/api/stat/go")
}

func TestRunTLSBase64Server(t *testing.T) {
	//nolint:lll // base64-encoded test certificate must remain on a single line
	cert := `LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUMrekNDQWVPZ0F3SUJBZ0lKQUxiWkVEdlVRckZLTUEwR0NTcUdTSWIzRFFFQkJRVUFNQlF4RWpBUUJnTlYKQkFNTUNXeHZZMkZzYUc5emREQWVGdzB4TmpBek1qZ3dNek13TkRGYUZ3MHlOakF6TWpZd016TXdOREZhTUJReApFakFRQmdOVkJBTU1DV3h2WTJGc2FHOXpkRENDQVNJd0RRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DCmdnRUJBTWoxK3hnNGpWTHpWbkI1ajduMXVsMzBXRUU0QkN6Y05GeGc1QU9CNUg1cSt3amUwWVlpVkZnNlBReXYKR0NpcHFJUlhWUmRWUTFoSFNldW5ZR0tlOGxxM1NiMVg4UFVKMTJ2OXVSYnBTOURLMU93cWs4cnNQRHU2c1ZUTApxS0tnSDFaOHlhenphUzBBYlh1QTVlOWdPL1J6aWpibnBFUCtxdU00ZHVlaU1QVkVKeUxxK0VvSVFZK01NOE1QCjhkWnpMNFhabDd3TDRVc0NON3JQY082VzN0bG5UMGlPM2g5Yy9ZbTJoRmh6K0tOSjlLUlJDdnRQR1pFU2lndEsKYkhzWEgwOTlXRG84di9XcDUvZXZCdy8rSkQwb3B4bUNmSElCQUxIdDl2NTNSdnZzRFoxdDMzUnB1NUM4em5FWQpZMkF5N05neGhxanFvV0pxQTQ4bEplQTBjbHNDQXdFQUFhTlFNRTR3SFFZRFZSME9CQllFRkMwYlRVMVhvZmVoCk5LSWVsYXNoSXNxS2lkRFlNQjhHQTFVZEl3UVlNQmFBRkMwYlRVMVhvZmVoTktJZWxhc2hJc3FLaWREWU1Bd0cKQTFVZEV3UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUZCUUFEZ2dFQkFBaUpMOElNVHdOWDlYcVFXWURGZ2tHNApBbnJWd1FocmVBcUM5clN4RENqcXFuTUhQSEd6Y0NlRE1MQU1vaDBrT3kyMG5vd1VHTnRDWjB1QnZuWDJxMWJOCmcxanQrR0JjTEpEUjNMTDRDcE5PbG0zWWhPeWN1TmZXTXhUQTdCWGttblNyWkQvN0toQXJzQkVZOGF1bHh3S0oKSFJnTmxJd2Uxb0ZEMVlkWDFCUzVwcDR0MjVCNlZxNEEzRk1NVWtWb1dFNjg4bkUxNjhodlFnd2pySGtnSGh3ZQplTjhsR0UyRGhGcmFYbldtRE1kd2FIRDNIUkZHaHlwcElGTitmN0JxYldYOWdNK1QyWVJUZk9iSVhMV2JxSkxECjNNay9Oa3hxVmNnNGVZNTR3SjF1ZkNVR0FZQUlhWTZmUXFpTlV6OG5od0szdDQ1TkJWVDl5L3VKWHFuVEx5WT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=`
	//nolint:lll // base64-encoded test private key must remain on a single line
	key := `LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBeVBYN0dEaU5Vdk5XY0htUHVmVzZYZlJZUVRnRUxOdzBYR0RrQTRIa2ZtcjdDTjdSCmhpSlVXRG85REs4WUtLbW9oRmRWRjFWRFdFZEo2NmRnWXA3eVdyZEp2VmZ3OVFuWGEvMjVGdWxMME1yVTdDcVQKeXV3OE83cXhWTXVvb3FBZlZuekpyUE5wTFFCdGU0RGw3MkE3OUhPS051ZWtRLzZxNHpoMjU2SXc5VVFuSXVyNApTZ2hCajR3end3L3gxbk12aGRtWHZBdmhTd0kzdXM5dzdwYmUyV2RQU0k3ZUgxejlpYmFFV0hQNG8wbjBwRkVLCiswOFprUktLQzBwc2V4Y2ZUMzFZT2p5Lzlhbm45NjhIRC80a1BTaW5HWUo4Y2dFQXNlMzIvbmRHKyt3Tm5XM2YKZEdtN2tMek9jUmhqWURMczJER0dxT3FoWW1vRGp5VWw0RFJ5V3dJREFRQUJBb0lCQUdUS3FzTjlLYlNmQTQycQpDcUkwVXVMb3VKTU5hMXFzbno1dUFpNllLV2dXZEE0QTQ0bXBFakNtRlJTVmhVSnZ4V3VLK2N5WUlRelh4SVdECkQxNm5aZHFGNzJBZUNXWjlKeVNzdnZaMDBHZktNM3kzNWlSeTA4c0pXZ096bWNMbkdKQ2lTZXlLc1FlM0hUSkMKZGhEWGJYcXZzSFRWUFpnMDFMVGVEeFVpVGZmVThOTUtxUjJBZWNRMnNURHdYRWhBblR5QXRuemwvWGFCZ0Z6dQpVNkc3RnpHTTV5OWJ4a2ZRVmt2eStERUprSEdOT2p6d2NWZkJ5eVZsNjEwaXhtRzF2bXhWajlQYldtSVBzVVY4CnlTbWpodkRRYk9mb3hXMGg5dlRsVHFHdFFjQnc5NjJvc25ERE1XRkNkTTdsek8wVDdSUm5QVkdJUnBDSk9LaHEKa2VxSEt3RUNnWUVBOHd3SS9pWnVnaG9UWFRORzlMblFRL1dBdHNxTzgwRWpNVFVoZW81STFrT3ptVXowOXB5aAppQXNVRG9OMC8yNnRaNVdOamxueVp1N2R2VGMveDNkVFpwbU5ub284Z2NWYlFORUNEUnpxZnVROVBQWG0xU041CjZwZUJxQXZCdjc4aGpWMDVhWHpQRy9WQmJlaWc3bDI5OUVhckVBK2Evb0gzS3JnRG9xVnFFMEVDZ1lFQTA2dkEKWUptZ2c0ZlpSdWNBWW9hWXNMejlaOXJDRmpUZTFQQlRtVUprYk9SOHZGSUhIVFRFV2kvU3V4WEwwd0RTZW9FMgo3QlFtODZnQ0M3L0tnUmRyem9CcVo1cVM5TXYyZHNMZ1k2MzVWU2dqamZaa1ZMaUgxVlJScFNRT2JZbmZveXNnCmdhdGNIU0tNRXhkNFNMUUJ5QXVJbVhQK0w1YXlEQmNFSmZicVNwc0NnWUI3OElzMWIwdXpOTERqT2g3WTlWaHIKRDJxUHpFT1JjSW9Oc2RaY3RPb1h1WGFBbW1uZ3lJYm01UjlaTjFnV1djNDdvRndMVjNyeFdxWGdzNmZtZzhjWAo3djMwOXZGY0M5UTQvVnhhYTRCNUxOSzluM2dUQUlCUFRPdGxVbmwrMm15MXRmQnRCcVJtMFc2SUtiVEhXUzVnCnZ4akVtL0NpRUl5R1VFZ3FUTWdIQVFLQmdCS3VYZFFvdXRuZzYzUXVmd0l6RHRiS1Z6TUxRNFhpTktobWJYcGgKT2F2Q25wK2dQYkIrTDdZbDhsdEFtVFNPSmdWWjBoY1QwRHhBMzYxWngrMk11NThHQmw0T2JsbmNobXdFMXZqMQpLY1F5UHJFUXhkb1VUeWlzd0dmcXZyczhKOWltdmIrejkvVTZUMUtBQjhXaTNXVmlYelByNE1zaWFhUlhnNjQyCkZJZHhBb0dBWjcvNzM1ZGtoSmN5T2ZzK0xLc0xyNjhKU3N0b29yWE9ZdmRNdTErSkdhOWlMdWhuSEVjTVZXQzgKSXVpaHpQZmxvWnRNYkdZa1pKbjhsM0JlR2Q4aG1mRnRnVGdaR1BvVlJldGZ0MkxERkxuUHhwMnNFSDVPRkxzUQpSK0sva0FPdWw4ZVN0V3VNWE9GQTlwTXpHa0dFZ0lGSk1KT3lhSk9OM2tlZFFJOGRlQ009Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==`
	cfg := initTest()

	cfg.Core.SSL = true
	cfg.Core.Port = "8089"
	cfg.Core.CertPath = ""
	cfg.Core.KeyPath = ""
	cfg.Core.CertBase64 = cert
	cfg.Core.KeyBase64 = key

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, RunHTTPServer(ctx, cfg, q))
	}()

	defer func() {
		// close the server
		cancel()
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	testRequest(t, "https://localhost:8089/api/stat/go")
}

func TestRunAutoTLSServer(t *testing.T) {
	cfg := initTest()
	cfg.Core.AutoTLS.Enabled = true
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, RunHTTPServer(ctx, cfg, q))
	}()

	defer func() {
		// close the server
		cancel()
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)
}

func TestLoadTLSCertError(t *testing.T) {
	cfg := initTest()

	cfg.Core.SSL = true
	cfg.Core.Port = "8087"
	cfg.Core.CertPath = "../config/config.yml"
	cfg.Core.KeyPath = "../config/config.yml"

	assert.Error(t, RunHTTPServer(context.Background(), cfg, q))
}

func TestMissingTLSCertcfgg(t *testing.T) {
	cfg := initTest()

	cfg.Core.SSL = true
	cfg.Core.Port = "8087"
	cfg.Core.CertPath = ""
	cfg.Core.KeyPath = ""
	cfg.Core.CertBase64 = ""
	cfg.Core.KeyBase64 = ""

	err := RunHTTPServer(context.Background(), cfg, q)
	require.Error(t, RunHTTPServer(context.Background(), cfg, q))
	assert.Equal(t, "missing https cert config", err.Error())
}

func TestRootHandler(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	// log for json
	cfg.Log.Format = "json"

	r.GET("/").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			data := r.Body.Bytes()

			value, _ := jsonparser.GetString(data, "text")

			assert.Equal(t, "Welcome to notification server.", value)
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestAPIStatusGoHandler(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	r.GET("/api/stat/go").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			data := r.Body.Bytes()

			value, _ := jsonparser.GetString(data, "go_version")

			assert.Equal(t, goVersion, value)
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestAPIStatusAppHandler(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	appVersion := "v1.0.0"
	SetVersion(appVersion)

	r.GET("/api/stat/app").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			data := r.Body.Bytes()

			value, _ := jsonparser.GetString(data, "version")

			assert.Equal(t, appVersion, value)
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestAPIConfigHandler(t *testing.T) {
	cfg := initTest()

	// Set sensitive values to verify they are redacted
	cfg.Android.Credential = "secret-fcm-credential"
	cfg.Ios.Password = "secret-ios-password"
	cfg.Stat.Redis.Password = "secret-redis-password"

	r := gofight.New()

	r.GET("/api/config").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)

			body := r.Body.String()
			assert.NotContains(t, body, "secret-fcm-credential")
			assert.NotContains(t, body, "secret-ios-password")
			assert.NotContains(t, body, "secret-redis-password")
			assert.Contains(t, body, "[REDACTED]")
		})
}

func TestMissingNotificationsParameter(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	// missing notifications parameter.
	r.POST("/api/push").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})
}

func TestEmptyNotifications(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	// notifications is empty.
	r.POST("/api/push").
		SetJSON(gofight.D{
			"notifications": []notify.PushNotification{},
		}).
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})
}

func TestMutableContent(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	// notifications is empty.
	r.POST("/api/push").
		SetJSON(gofight.D{
			"notifications": []gofight.D{
				{
					"tokens":          []string{"aaaaa", "bbbbb"},
					"platform":        core.PlatFormAndroid,
					"message":         "Welcome From API",
					"mutable_content": 1,
					"topic":           "test",
					"badge":           1,
					"alert": gofight.D{
						"title": "title",
						"body":  "body",
					},
				},
			},
		}).
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			// json: cannot unmarshal number into Go struct field notify.PushNotification.mutable_content of type bool
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})
}

func TestOutOfRangeMaxNotifications(t *testing.T) {
	cfg := initTest()

	cfg.Core.MaxNotification = int64(1)

	r := gofight.New()

	// notifications is empty.
	r.POST("/api/push").
		SetJSON(gofight.D{
			"notifications": []gofight.D{
				{
					"tokens":   []string{"aaaaa", "bbbbb"},
					"platform": core.PlatFormAndroid,
					"message":  "Welcome API From Android",
				},
				{
					"tokens":   []string{"aaaaa", "bbbbb"},
					"platform": core.PlatFormAndroid,
					"message":  "Welcome API From Android",
				},
			},
		}).
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})
}

func TestSuccessPushHandler(t *testing.T) {
	t.Skip()
	cfg := initTest()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	r := gofight.New()

	r.POST("/api/push").
		SetJSON(gofight.D{
			"notifications": []gofight.D{
				{
					"tokens":   []string{androidToken, "bbbbb"},
					"platform": core.PlatFormAndroid,
					"message":  "Welcome Android",
				},
			},
		}).
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestSysStatsHandler(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	r.GET("/sys/stats").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestMetricsHandler(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	r.GET("/metrics").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestGETHeartbeatHandler(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	r.GET("/healthz").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestHEADHeartbeatHandler(t *testing.T) {
	cfg := initTest()

	r := gofight.New()

	r.HEAD("/healthz").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestVersionHandler(t *testing.T) {
	SetVersion("3.0.0")
	cfg := initTest()

	r := gofight.New()

	r.GET("/version").
		Run(routerEngine(cfg, q), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			data := r.Body.Bytes()

			value, _ := jsonparser.GetString(data, "version")

			assert.Equal(t, "3.0.0", value)
		})
}

func TestDisabledHTTPServer(t *testing.T) {
	cfg := initTest()
	cfg.Core.Enabled = false
	err := RunHTTPServer(context.Background(), cfg, q)
	cfg.Core.Enabled = true

	require.NoError(t, err)
}

func TestSenMultipleNotifications(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	cfg.Ios.Enabled = true
	cfg.Ios.KeyPath = testKeyPath
	err := notify.InitAPNSClient(ctx, cfg)
	require.NoError(t, err)

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := notify.RequestPush{
		Notifications: []notify.PushNotification{
			// ios
			{
				Tokens: []string{
					"11aa01229f15f0f0c52029d8cf8cd0aeaf2365fe4cebc4af26cd6d76b7919ef7",
				},
				Platform: core.PlatFormIos,
				Message:  "Welcome iOS",
			},
			// android
			{
				Tokens:   []string{androidToken, "bbbbb"},
				Platform: core.PlatFormAndroid,
				Message:  "Welcome Android",
			},
		},
	}

	count, logs := handleNotification(ctx, cfg, req, q)
	assert.Equal(t, 3, count)
	assert.Empty(t, logs)
}

func TestDisabledAndroidNotifications(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	cfg.Ios.Enabled = true
	cfg.Ios.KeyPath = testKeyPath
	err := notify.InitAPNSClient(ctx, cfg)
	require.NoError(t, err)

	cfg.Android.Enabled = false
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := notify.RequestPush{
		Notifications: []notify.PushNotification{
			// ios
			{
				Tokens: []string{
					"11aa01229f15f0f0c5209d8cf8cd0aeaf2365fe4cebc4af26cd6d76b7919ef7",
				},
				Platform: core.PlatFormIos,
				Message:  "Welcome iOS",
			},
			// android
			{
				Tokens:   []string{androidToken, "bbbbb"},
				Platform: core.PlatFormAndroid,
				Message:  "Welcome Android",
			},
		},
	}

	count, logs := handleNotification(ctx, cfg, req, q)
	assert.Equal(t, 1, count)
	assert.Empty(t, logs)
}

func TestSyncModeForNotifications(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	cfg.Ios.Enabled = true
	cfg.Ios.KeyPath = testKeyPath
	err := notify.InitAPNSClient(ctx, cfg)
	require.NoError(t, err)

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	// enable sync mode
	cfg.Core.Sync = true

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := notify.RequestPush{
		Notifications: []notify.PushNotification{
			// ios
			{
				Tokens: []string{
					"11aa01229f15f0f0c12029d8c111d1ae1f2365f14cebc4af26cd6d76b7919ef7",
				},
				Platform: core.PlatFormIos,
				Message:  "Welcome iOS Sync",
			},
			// android
			{
				Tokens:   []string{androidToken, "bbbbb"},
				Platform: core.PlatFormAndroid,
				Message:  "Welcome Android Sync",
			},
		},
	}

	count, logs := handleNotification(ctx, cfg, req, q)
	assert.Equal(t, 3, count)
	assert.Len(t, logs, 3)
}

func TestSyncModeForTopicNotification(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")
	cfg.Log.HideToken = false

	// enable sync mode
	cfg.Core.Sync = true

	req := notify.RequestPush{
		Notifications: []notify.PushNotification{
			// android
			{
				// error:InvalidParameters
				// Check that the provided parameters have the right name and type.
				Topic:    "/topics/foo-bar@@@##",
				Platform: core.PlatFormAndroid,
				Message:  "This is a Firebase Cloud Messaging Topic Message!",
			},
			// android
			{
				// success
				Topic:    "/topics/foo-bar",
				Platform: core.PlatFormAndroid,
				Message:  "This is a Firebase Cloud Messaging Topic Message!",
			},
			// android
			{
				// success
				Condition: "'dogs' in topics || 'cats' in topics",
				Platform:  core.PlatFormAndroid,
				Message:   "This is a Firebase Cloud Messaging Topic Message!",
			},
		},
	}

	count, logs := handleNotification(ctx, cfg, req, q)
	assert.Equal(t, 2, count)
	assert.Empty(t, logs)
}

func TestSyncModeForDeviceGroupNotification(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")
	cfg.Log.HideToken = false

	// enable sync mode
	cfg.Core.Sync = true

	req := notify.RequestPush{
		Notifications: []notify.PushNotification{
			// android
			{
				Topic:    "aUniqueKey",
				Platform: core.PlatFormAndroid,
				Message:  "This is a Firebase Cloud Messaging Device Group Message!",
			},
		},
	}

	// success
	count, logs := handleNotification(ctx, cfg, req, q)
	assert.Equal(t, 1, count)
	assert.Empty(t, logs)
}

func TestDisabledIosNotifications(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	cfg.Ios.Enabled = false
	cfg.Ios.KeyPath = testKeyPath
	err := notify.InitAPNSClient(ctx, cfg)
	require.NoError(t, err)

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := notify.RequestPush{
		Notifications: []notify.PushNotification{
			// ios
			{
				Tokens: []string{
					"11aa01229f15f0f0c52021d8cf3cd0ae1f2365fe4cebc4af26cd6d76b7919ef7",
				},
				Platform: core.PlatFormIos,
				Message:  "Welcome iOS platform",
			},
			// android
			{
				Tokens:   []string{androidToken, androidToken + "_"},
				Platform: core.PlatFormAndroid,
				Message:  "Welcome Android platform",
			},
		},
	}

	count, logs := handleNotification(ctx, cfg, req, q)
	assert.Equal(t, 2, count)
	assert.Empty(t, logs)
}

// Tests for refactored helper functions

func TestIsPlatformEnabled(t *testing.T) {
	cfg := initTest()

	tests := []struct {
		name           string
		platform       int
		iosEnabled     bool
		androidEnabled bool
		huaweiEnabled  bool
		want           bool
	}{
		{
			name:       "iOS enabled",
			platform:   core.PlatFormIos,
			iosEnabled: true,
			want:       true,
		},
		{
			name:       "iOS disabled",
			platform:   core.PlatFormIos,
			iosEnabled: false,
			want:       false,
		},
		{
			name:           "Android enabled",
			platform:       core.PlatFormAndroid,
			androidEnabled: true,
			want:           true,
		},
		{
			name:           "Android disabled",
			platform:       core.PlatFormAndroid,
			androidEnabled: false,
			want:           false,
		},
		{
			name:          "Huawei enabled",
			platform:      core.PlatFormHuawei,
			huaweiEnabled: true,
			want:          true,
		},
		{
			name:          "Huawei disabled",
			platform:      core.PlatFormHuawei,
			huaweiEnabled: false,
			want:          false,
		},
		{
			name:     "Unknown platform",
			platform: 99,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Ios.Enabled = tt.iosEnabled
			cfg.Android.Enabled = tt.androidEnabled
			cfg.Huawei.Enabled = tt.huaweiEnabled

			got := isPlatformEnabled(cfg, tt.platform)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterEnabledNotifications(t *testing.T) {
	cfg := initTest()
	cfg.Ios.Enabled = true
	cfg.Android.Enabled = true
	cfg.Huawei.Enabled = false

	notifications := []notify.PushNotification{
		{Platform: core.PlatFormIos, Message: "iOS message"},
		{Platform: core.PlatFormAndroid, Message: "Android message"},
		{Platform: core.PlatFormHuawei, Message: "Huawei message"},
		{Platform: core.PlatFormIos, Message: "iOS message 2"},
	}

	result := filterEnabledNotifications(cfg, notifications)

	assert.Len(t, result, 3)
	assert.Equal(t, "iOS message", result[0].Message)
	assert.Equal(t, "Android message", result[1].Message)
	assert.Equal(t, "iOS message 2", result[2].Message)
}

func TestFilterEnabledNotificationsAllDisabled(t *testing.T) {
	cfg := initTest()
	cfg.Ios.Enabled = false
	cfg.Android.Enabled = false
	cfg.Huawei.Enabled = false

	notifications := []notify.PushNotification{
		{Platform: core.PlatFormIos, Message: "iOS message"},
		{Platform: core.PlatFormAndroid, Message: "Android message"},
	}

	result := filterEnabledNotifications(cfg, notifications)

	assert.Empty(t, result)
}

func TestCountNotificationTargets(t *testing.T) {
	tests := []struct {
		name         string
		notification *notify.PushNotification
		want         int
	}{
		{
			name: "tokens only",
			notification: &notify.PushNotification{
				Tokens: []string{"token1", "token2", "token3"},
			},
			want: 3,
		},
		{
			name: "topic only",
			notification: &notify.PushNotification{
				Topic: "test-topic",
			},
			want: 1,
		},
		{
			name: "tokens and topic",
			notification: &notify.PushNotification{
				Tokens: []string{"token1", "token2"},
				Topic:  "test-topic",
			},
			want: 3,
		},
		{
			name:         "empty notification",
			notification: &notify.PushNotification{},
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countNotificationTargets(tt.notification)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestHandleNotificationDoesNotMutateConfigSync verifies that handling a request
// never writes back to the shared cfg.Core.Sync. Previously a non-local-queue
// request flipped cfg.Core.Sync to false, leaking into concurrent and
// subsequent requests.
func TestHandleNotificationDoesNotMutateConfigSync(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	// Sync requested, but the configured queue is a non-local (external) engine,
	// for which sync mode cannot apply.
	cfg.Core.Sync = true
	cfg.Queue.Engine = string(core.NSQ)

	// Disable every platform so no notification is actually dispatched. This
	// keeps the test free of any network or credential dependency while still
	// exercising the sync-mode decision.
	cfg.Ios.Enabled = false
	cfg.Android.Enabled = false
	cfg.Huawei.Enabled = false

	req := notify.RequestPush{
		Notifications: []notify.PushNotification{
			{
				Tokens:   []string{"token"},
				Platform: core.PlatFormIos,
				Message:  "should not be sent",
			},
		},
	}

	count, logs := handleNotification(ctx, cfg, req, q)

	assert.Equal(t, 0, count)
	assert.Empty(t, logs)
	assert.True(
		t,
		cfg.Core.Sync,
		"handleNotification must not mutate the shared cfg.Core.Sync",
	)
}

// TestSyncModeAggregatesLogsConcurrently exercises the synchronous local-queue
// path with many notifications processed by concurrent workers and asserts that
// every log is collected (no lost updates) and that aggregation preserves
// request order. Run with `-race` to detect concurrent access to shared state.
func TestSyncModeAggregatesLogsConcurrently(t *testing.T) {
	ctx := context.Background()
	cfg := initTest()

	// Local queue + sync mode selects the synchronous dispatch path.
	cfg.Queue.Engine = string(core.LocalQueue)
	cfg.Core.Sync = true
	cfg.Ios.Enabled = true

	// Replace the real sender with a deterministic stub so the test does not
	// depend on any push backend. The sleep widens the window in which the
	// queue workers run concurrently, surfacing data races under `go test -race`.
	original := sendNotification
	defer func() { sendNotification = original }()
	sendNotification = func(
		_ context.Context, msg qcore.TaskMessage, _ *config.ConfYaml,
	) (*notify.ResponsePush, error) {
		n, ok := msg.(*notify.PushNotification)
		require.True(t, ok)
		time.Sleep(time.Millisecond)
		entries := make([]logx.LogPushEntry, 0, len(n.Tokens))
		for _, token := range n.Tokens {
			entries = append(entries, logx.LogPushEntry{Token: token, Message: n.Message})
		}
		return &notify.ResponsePush{Logs: entries}, nil
	}

	// A dedicated pool with several workers guarantees real concurrency
	// regardless of the host's GOMAXPROCS / CPU count.
	pool := queue.NewPool(
		4,
		queue.WithFn(func(_ context.Context, _ qcore.TaskMessage) error { return nil }),
		queue.WithLogger(logx.QueueLogger()),
	)
	defer pool.Release()

	const total = 50
	notifications := make([]notify.PushNotification, 0, total)
	for i := range total {
		notifications = append(notifications, notify.PushNotification{
			Tokens:   []string{fmt.Sprintf("token-%d", i)},
			Platform: core.PlatFormIos,
			Message:  fmt.Sprintf("message-%d", i),
		})
	}

	count, logs := handleNotification(
		ctx, cfg, notify.RequestPush{Notifications: notifications}, pool,
	)

	assert.Equal(t, total, count)
	require.Len(t, logs, total)
	// Each task writes to its own slot, flattened in order, so logs must come
	// back in request order.
	for i := range total {
		assert.Equal(t, fmt.Sprintf("token-%d", i), logs[i].Token)
	}
}

// TestSyncModeEnqueueErrorDoesNotDeadlock verifies that when QueueTask fails to
// schedule a task (here, because the pool is already released), the wait group
// is still released so the request returns instead of hanging, and a failure
// log is recorded for the affected notification.
func TestSyncModeEnqueueErrorDoesNotDeadlock(t *testing.T) {
	cfg := initTest()
	cfg.Ios.Enabled = true

	// Stub the sender so that even if a task were unexpectedly scheduled, it
	// would not touch any push backend and would yield one log entry per token,
	// matching the failure path's shape.
	original := sendNotification
	defer func() { sendNotification = original }()
	sendNotification = func(
		_ context.Context, msg qcore.TaskMessage, _ *config.ConfYaml,
	) (*notify.ResponsePush, error) {
		n := msg.(*notify.PushNotification)
		entries := make([]logx.LogPushEntry, 0, len(n.Tokens))
		for _, token := range n.Tokens {
			entries = append(entries, logx.LogPushEntry{Token: token})
		}
		return &notify.ResponsePush{Logs: entries}, nil
	}

	// A released pool rejects QueueTask, exercising the enqueue-error branch.
	pool := queue.NewPool(
		1,
		queue.WithFn(func(_ context.Context, _ qcore.TaskMessage) error { return nil }),
		queue.WithLogger(logx.QueueLogger()),
	)
	pool.Release()

	notifications := []*notify.PushNotification{
		{Tokens: []string{"a"}, Platform: core.PlatFormIos, Message: "x"},
		{Tokens: []string{"b"}, Platform: core.PlatFormIos, Message: "y"},
	}

	done := make(chan []logx.LogPushEntry, 1)
	go func() {
		done <- pushNotificationsSync(cfg, notifications, pool)
	}()

	select {
	case logs := <-done:
		// One failure log entry per token across both notifications.
		assert.Len(t, logs, 2)
	case <-time.After(2 * time.Second):
		t.Fatal("pushNotificationsSync deadlocked when QueueTask returned an error")
	}
}
