package config

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test file is missing
func TestMissingFile(t *testing.T) {
	filename := "nonexistent_file.yml"
	conf, err := LoadConf(filename)

	assert.Nil(t, conf)
	require.Error(t, err)
	// Check that the error message includes the filename and describes the issue
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Contains(t, err.Error(), filename)
}

// Test invalid YAML content
func TestInvalidYAMLFile(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpFile := "test_invalid.yml"
	content := []byte("invalid: yaml: content: [unclosed")

	// Write invalid content to a temporary file
	err := os.WriteFile(tmpFile, content, 0o600)
	require.NoError(t, err)
	defer os.Remove(tmpFile) // Clean up

	conf, err := LoadConf(tmpFile)

	assert.Nil(t, conf)
	require.Error(t, err)
	// Check that the error message includes the filename and describes parsing failure
	assert.Contains(t, err.Error(), "failed to parse config file")
	assert.Contains(t, err.Error(), tmpFile)
}

func TestEmptyConfig(t *testing.T) {
	conf, err := LoadConf("testdata/empty.yml")
	if err != nil {
		panic("failed to load config.yml from file")
	}

	assert.Equal(t, uint(100), conf.Ios.MaxConcurrentPushes)
}

type ConfigTestSuite struct {
	suite.Suite
	ConfGorushDefault *ConfYaml
	ConfGorush        *ConfYaml
}

func (suite *ConfigTestSuite) SetupTest() {
	var err error
	suite.ConfGorushDefault, err = LoadConf()
	if err != nil {
		panic("failed to load default config.yml")
	}
	suite.ConfGorush, err = LoadConf("testdata/config.yml")
	if err != nil {
		panic("failed to load config.yml from file")
	}
}

// assertCommonConfig verifies config values shared by both default and file-loaded configs.
func (suite *ConfigTestSuite) assertCommonConfig(conf *ConfYaml) {
	// Core
	suite.Equal("8088", conf.Core.Port)
	suite.Equal(int64(30), conf.Core.ShutdownTimeout)
	suite.True(conf.Core.Enabled)
	suite.Equal(int64(runtime.NumCPU()), conf.Core.WorkerNum)
	suite.Equal(int64(8192), conf.Core.QueueNum)
	suite.Equal("release", conf.Core.Mode)
	suite.False(conf.Core.Sync)
	suite.Empty(conf.Core.FeedbackURL)
	suite.Equal(int64(10), conf.Core.FeedbackTimeout)
	suite.False(conf.Core.SSL)
	suite.Equal("cert.pem", conf.Core.CertPath)
	suite.Equal("key.pem", conf.Core.KeyPath)
	suite.Empty(conf.Core.CertBase64)
	suite.Empty(conf.Core.KeyBase64)
	suite.Equal(int64(100), conf.Core.MaxNotification)
	suite.Empty(conf.Core.HTTPProxy)

	// PID
	suite.False(conf.Core.PID.Enabled)
	suite.Equal("gorush.pid", conf.Core.PID.Path)
	suite.True(conf.Core.PID.Override)
	suite.False(conf.Core.AutoTLS.Enabled)
	suite.Equal(".cache", conf.Core.AutoTLS.Folder)
	suite.Empty(conf.Core.AutoTLS.Host)

	// API
	suite.Equal("/api/push", conf.API.PushURI)
	suite.Equal("/api/stat/go", conf.API.StatGoURI)
	suite.Equal("/api/stat/app", conf.API.StatAppURI)
	suite.Equal("/api/config", conf.API.ConfigURI)
	suite.Equal("/sys/stats", conf.API.SysStatURI)
	suite.Equal("/metrics", conf.API.MetricURI)
	suite.Equal("/healthz", conf.API.HealthURI)

	// iOS (common fields)
	suite.False(conf.Ios.Enabled)
	suite.Empty(conf.Ios.KeyBase64)
	suite.Equal("pem", conf.Ios.KeyType)
	suite.Empty(conf.Ios.Password)
	suite.False(conf.Ios.Production)
	suite.Equal(uint(100), conf.Ios.MaxConcurrentPushes)
	suite.Equal(0, conf.Ios.MaxRetry)
	suite.Empty(conf.Ios.KeyID)
	suite.Empty(conf.Ios.TeamID)

	// Log
	suite.Equal("string", conf.Log.Format)
	suite.Equal("stdout", conf.Log.AccessLog)
	suite.Equal("debug", conf.Log.AccessLevel)
	suite.Equal("stderr", conf.Log.ErrorLog)
	suite.Equal("error", conf.Log.ErrorLevel)
	suite.True(conf.Log.HideToken)

	// Stat
	suite.Equal("memory", conf.Stat.Engine)
	suite.False(conf.Stat.Redis.Cluster)
	suite.Equal("localhost:6379", conf.Stat.Redis.Addr)
	suite.Empty(conf.Stat.Redis.Username)
	suite.Empty(conf.Stat.Redis.Password)
	suite.Equal(0, conf.Stat.Redis.DB)
	suite.Equal("bolt.db", conf.Stat.BoltDB.Path)
	suite.Equal("gorush", conf.Stat.BoltDB.Bucket)
	suite.Equal("bunt.db", conf.Stat.BuntDB.Path)
	suite.Equal("level.db", conf.Stat.LevelDB.Path)
	suite.Equal("badger.db", conf.Stat.BadgerDB.Path)

	// gRPC
	suite.False(conf.GRPC.Enabled)
	suite.Equal("9000", conf.GRPC.Port)
}

func (suite *ConfigTestSuite) TestValidateConfDefault() {
	suite.assertCommonConfig(suite.ConfGorushDefault)

	// Default-specific assertions
	suite.Empty(suite.ConfGorushDefault.Core.Address)
	suite.Empty(suite.ConfGorushDefault.Core.FeedbackHeader)
	suite.False(suite.ConfGorushDefault.Log.HideMessages)

	// Android defaults
	suite.True(suite.ConfGorushDefault.Android.Enabled)
	suite.Empty(suite.ConfGorushDefault.Android.KeyPath)
	suite.Empty(suite.ConfGorushDefault.Android.Credential)
	suite.Equal(0, suite.ConfGorushDefault.Android.MaxRetry)

	// iOS defaults
	suite.Empty(suite.ConfGorushDefault.Ios.KeyPath)

	// Queue defaults
	suite.Equal("local", suite.ConfGorushDefault.Queue.Engine)
	suite.Equal("127.0.0.1:4150", suite.ConfGorushDefault.Queue.NSQ.Addr)
	suite.Equal("gorush", suite.ConfGorushDefault.Queue.NSQ.Topic)
	suite.Equal("gorush", suite.ConfGorushDefault.Queue.NSQ.Channel)

	suite.Equal("127.0.0.1:4222", suite.ConfGorushDefault.Queue.NATS.Addr)
	suite.Equal("gorush", suite.ConfGorushDefault.Queue.NATS.Subj)
	suite.Equal("gorush", suite.ConfGorushDefault.Queue.NATS.Queue)

	suite.Equal("127.0.0.1:6379", suite.ConfGorushDefault.Queue.Redis.Addr)
	suite.Equal("gorush", suite.ConfGorushDefault.Queue.Redis.StreamName)
	suite.Equal("gorush", suite.ConfGorushDefault.Queue.Redis.Group)
	suite.Equal("gorush", suite.ConfGorushDefault.Queue.Redis.Consumer)
	suite.Empty(suite.ConfGorushDefault.Queue.Redis.Username)
	suite.Empty(suite.ConfGorushDefault.Queue.Redis.Password)
	suite.False(suite.ConfGorushDefault.Queue.Redis.WithTLS)
	suite.Equal(0, suite.ConfGorushDefault.Queue.Redis.DB)
}

func (suite *ConfigTestSuite) TestValidateConf() {
	suite.assertCommonConfig(suite.ConfGorush)

	// File-specific assertions
	suite.Len(suite.ConfGorush.Core.FeedbackHeader, 1)
	suite.Equal(
		"x-gorush-token:4e989115e09680f44a645519fed6a976",
		suite.ConfGorush.Core.FeedbackHeader[0],
	)

	// Android from file
	suite.True(suite.ConfGorush.Android.Enabled)
	suite.Equal("key.json", suite.ConfGorush.Android.KeyPath)
	suite.Equal("CREDENTIAL_JSON_DATA", suite.ConfGorush.Android.Credential)
	suite.Equal(0, suite.ConfGorush.Android.MaxRetry)

	// iOS from file
	suite.Equal("key.pem", suite.ConfGorush.Ios.KeyPath)
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("GORUSH_CORE_PORT", "9001")
	t.Setenv("GORUSH_GRPC_ENABLED", "true")
	t.Setenv("GORUSH_CORE_MAX_NOTIFICATION", "200")
	t.Setenv("GORUSH_IOS_KEY_ID", "ABC123DEFG")
	t.Setenv("GORUSH_IOS_TEAM_ID", "DEF123GHIJ")
	t.Setenv("GORUSH_API_HEALTH_URI", "/healthz")
	t.Setenv("GORUSH_CORE_FEEDBACK_HOOK_URL", "http://example.com")
	t.Setenv("GORUSH_CORE_FEEDBACK_HEADER", "x-api-key:1234567890 x-auth-key:0987654321")

	ConfGorush, err := LoadConf("testdata/config.yml")
	if err != nil {
		panic("failed to load config.yml from file")
	}
	assert.Equal(t, "9001", ConfGorush.Core.Port)
	assert.Equal(t, int64(200), ConfGorush.Core.MaxNotification)
	assert.True(t, ConfGorush.GRPC.Enabled)
	assert.Equal(t, "ABC123DEFG", ConfGorush.Ios.KeyID)
	assert.Equal(t, "DEF123GHIJ", ConfGorush.Ios.TeamID)
	assert.Equal(t, "/healthz", ConfGorush.API.HealthURI)
	assert.Equal(t, "http://example.com", ConfGorush.Core.FeedbackURL)
	assert.Equal(t, "x-api-key:1234567890", ConfGorush.Core.FeedbackHeader[0])
	assert.Equal(t, "x-auth-key:0987654321", ConfGorush.Core.FeedbackHeader[1])
}

func TestRedisDBConfiguration(t *testing.T) {
	// Test loading Redis DB configuration from file
	conf, err := LoadConf("testdata/redis_db_config.yml")
	if err != nil {
		t.Fatalf("failed to load redis_db_config.yml: %v", err)
	}

	// Test queue.redis.db is properly loaded
	assert.Equal(t, "redis", conf.Queue.Engine)
	assert.Equal(t, 5, conf.Queue.Redis.DB)

	// Test stat.redis.db is properly loaded
	assert.Equal(t, "redis", conf.Stat.Engine)
	assert.Equal(t, 3, conf.Stat.Redis.DB)
}

func TestRedisDBConfigurationFromEnv(t *testing.T) {
	// Test loading Redis DB configuration from environment variables
	t.Setenv("GORUSH_QUEUE_REDIS_DB", "7")
	t.Setenv("GORUSH_STAT_REDIS_DB", "9")

	conf, err := LoadConf()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Test queue.redis.db is properly loaded from env
	assert.Equal(t, 7, conf.Queue.Redis.DB)

	// Test stat.redis.db is properly loaded from env
	assert.Equal(t, 9, conf.Stat.Redis.DB)
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty port should be valid",
			port:    "",
			wantErr: false,
		},
		{
			name:    "valid port 80",
			port:    "80",
			wantErr: false,
		},
		{
			name:    "valid port 8080",
			port:    "8080",
			wantErr: false,
		},
		{
			name:    "valid port 65535",
			port:    "65535",
			wantErr: false,
		},
		{
			name:    "valid port 1",
			port:    "1",
			wantErr: false,
		},
		{
			name:    "invalid port 0",
			port:    "0",
			wantErr: true,
			errMsg:  "port out of range",
		},
		{
			name:    "invalid port 65536",
			port:    "65536",
			wantErr: true,
			errMsg:  "port out of range",
		},
		{
			name:    "invalid port format",
			port:    "abc",
			wantErr: true,
			errMsg:  "invalid port format",
		},
		{
			name:    "invalid port with injection",
			port:    "80;rm -rf /",
			wantErr: true,
			errMsg:  "invalid port format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePort() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePort() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("ValidatePort() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty address should be valid",
			addr:    "",
			wantErr: false,
		},
		{
			name:    "valid IPv4 localhost",
			addr:    "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "valid IPv4 all interfaces",
			addr:    "0.0.0.0",
			wantErr: false,
		},
		{
			name:    "valid IPv6 localhost",
			addr:    "::1",
			wantErr: false,
		},
		{
			name:    "valid hostname",
			addr:    "localhost",
			wantErr: false,
		},
		{
			name:    "invalid address too long",
			addr:    strings.Repeat("a", 254),
			wantErr: true,
			errMsg:  "invalid address format",
		},
		{
			name:    "invalid address with double dots",
			addr:    "test..example.com",
			wantErr: true,
			errMsg:  "invalid address format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddress(tt.addr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAddress() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf(
						"ValidateAddress() error = %v, want error containing %v",
						err,
						tt.errMsg,
					)
				}
			} else if err != nil {
				t.Errorf("ValidateAddress() error = %v, want nil", err)
			}
		})
	}
}

func TestValidatePIDPath(t *testing.T) {
	tests := []struct {
		name    string
		pidPath string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path should be valid",
			pidPath: "",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			pidPath: "gorush.pid",
			wantErr: false,
		},
		{
			name:    "valid absolute path in tmp",
			pidPath: "/tmp/gorush.pid",
			wantErr: false,
		},
		{
			name:    "valid absolute path with .. components (cleaned)",
			pidPath: "/tmp/foo/../gorush.pid",
			wantErr: false,
		},
		{
			name:    "valid absolute path with multiple .. components",
			pidPath: "/home/user/subdir/../gorush.pid",
			wantErr: false,
		},
		{
			name:    "relative path traversal attack",
			pidPath: "../../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal detected",
		},
		{
			name:    "simple relative traversal",
			pidPath: "../gorush.pid",
			wantErr: true,
			errMsg:  "path traversal detected",
		},
		{
			name:    "complex relative traversal",
			pidPath: "subdir/../../gorush.pid",
			wantErr: true,
			errMsg:  "path traversal detected",
		},
		{
			name:    "attempt to write to /etc",
			pidPath: "/etc/gorush.pid",
			wantErr: true,
			errMsg:  "sensitive directory",
		},
		{
			name:    "attempt to write to /usr",
			pidPath: "/usr/bin/gorush.pid",
			wantErr: true,
			errMsg:  "sensitive directory",
		},
		{
			name:    "attempt to write to /var",
			pidPath: "/var/log/gorush.pid",
			wantErr: true,
			errMsg:  "sensitive directory",
		},
		{
			name:    "attempt to write to /sys",
			pidPath: "/sys/gorush.pid",
			wantErr: true,
			errMsg:  "sensitive directory",
		},
		{
			name:    "attempt to write to /proc",
			pidPath: "/proc/gorush.pid",
			wantErr: true,
			errMsg:  "sensitive directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePIDPath(tt.pidPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePIDPath() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf(
						"ValidatePIDPath() error = %v, want error containing %v",
						err,
						tt.errMsg,
					)
				}
			} else if err != nil {
				t.Errorf("ValidatePIDPath() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ConfYaml
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &ConfYaml{
				Core: SectionCore{
					Port:            "8088",
					Address:         "0.0.0.0",
					Mode:            "release",
					WorkerNum:       4,
					QueueNum:        8192,
					MaxNotification: 100,
				},
				Log: SectionLog{
					AccessLevel: "debug",
					ErrorLevel:  "error",
				},
				Stat:  SectionStat{Engine: "memory"},
				Queue: SectionQueue{Engine: "local"},
			},
			wantErr: false,
		},
		{
			name: "invalid port in config",
			cfg: &ConfYaml{
				Core: SectionCore{
					Port: "99999",
				},
			},
			wantErr: true,
			errMsg:  "invalid core port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateConfig() expected error but got none")
					return
				}

				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf(
						"ValidateConfig() error = %v, want error containing %v",
						err,
						tt.errMsg,
					)
				}
			} else if err != nil {
				t.Errorf("ValidateConfig() error = %v, want nil", err)
			}
		})
	}
}

// Benchmark tests for security validation functions
func BenchmarkValidatePort(b *testing.B) {
	for b.Loop() {
		_ = ValidatePort("8080")
	}
}

func BenchmarkValidateAddress(b *testing.B) {
	for b.Loop() {
		_ = ValidateAddress("127.0.0.1")
	}
}

func BenchmarkValidatePIDPath(b *testing.B) {
	for b.Loop() {
		_ = ValidatePIDPath("/tmp/gorush.pid")
	}
}

// Integration test for security validation
func TestSecurityValidationIntegration(t *testing.T) {
	// Test that all validation functions work together
	t.Run("complete security validation", func(t *testing.T) {
		// Test valid inputs
		if err := ValidatePort("8088"); err != nil {
			t.Errorf("Valid port should not error: %v", err)
		}

		if err := ValidateAddress("127.0.0.1"); err != nil {
			t.Errorf("Valid address should not error: %v", err)
		}

		if err := ValidatePIDPath("/tmp/test.pid"); err != nil {
			t.Errorf("Valid PID path should not error: %v", err)
		}

		// Test malicious inputs
		if err := ValidatePort("8080; rm -rf /"); err == nil {
			t.Error("Malicious port should error")
		}

		if err := ValidatePIDPath("../../../etc/passwd"); err == nil {
			t.Error("Path traversal should error")
		}

		if err := ValidatePIDPath("/etc/malicious.pid"); err == nil {
			t.Error("Sensitive directory write should error")
		}
	})
}

func TestAPIDefaultsFromEnv(t *testing.T) {
	// Test that API endpoint defaults can be overridden by environment variables
	t.Setenv("GORUSH_API_PUSH_URI", "/custom/push")
	t.Setenv("GORUSH_API_STAT_GO_URI", "/custom/stat/go")
	t.Setenv("GORUSH_API_METRIC_URI", "/custom/metrics")

	conf, err := LoadConf()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	assert.Equal(t, "/custom/push", conf.API.PushURI)
	assert.Equal(t, "/custom/stat/go", conf.API.StatGoURI)
	assert.Equal(t, "/custom/metrics", conf.API.MetricURI)
}

func TestLogDefaultsFromEnv(t *testing.T) {
	// Test that log level defaults can be overridden by environment variables
	t.Setenv("GORUSH_LOG_ACCESS_LEVEL", "info")
	t.Setenv("GORUSH_LOG_ERROR_LEVEL", "warn")
	t.Setenv("GORUSH_LOG_FORMAT", "json")

	conf, err := LoadConf()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	assert.Equal(t, "info", conf.Log.AccessLevel)
	assert.Equal(t, "warn", conf.Log.ErrorLevel)
	assert.Equal(t, "json", conf.Log.Format)
}

func TestLogLevelDefaultsWhenEmpty(t *testing.T) {
	// Test that when no log level is specified, defaults are used
	// This was the original bug - empty log levels caused initialization to fail
	conf, err := LoadConf()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify defaults are set
	assert.NotEmpty(t, conf.Log.AccessLevel, "access level should have default value")
	assert.NotEmpty(t, conf.Log.ErrorLevel, "error level should have default value")
	assert.Equal(t, "debug", conf.Log.AccessLevel)
	assert.Equal(t, "error", conf.Log.ErrorLevel)
}

func TestAllAPIEndpointsHaveDefaults(t *testing.T) {
	// Test that all API endpoints have proper defaults
	conf, err := LoadConf()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify all API endpoints have non-empty defaults
	assert.NotEmpty(t, conf.API.PushURI, "push_uri should have default")
	assert.NotEmpty(t, conf.API.StatGoURI, "stat_go_uri should have default")
	assert.NotEmpty(t, conf.API.StatAppURI, "stat_app_uri should have default")
	assert.NotEmpty(t, conf.API.ConfigURI, "config_uri should have default")
	assert.NotEmpty(t, conf.API.SysStatURI, "sys_stat_uri should have default")
	assert.NotEmpty(t, conf.API.MetricURI, "metric_uri should have default")
	assert.NotEmpty(t, conf.API.HealthURI, "health_uri should have default")

	// Verify correct default values
	assert.Equal(t, "/api/push", conf.API.PushURI)
	assert.Equal(t, "/api/stat/go", conf.API.StatGoURI)
	assert.Equal(t, "/api/stat/app", conf.API.StatAppURI)
	assert.Equal(t, "/api/config", conf.API.ConfigURI)
	assert.Equal(t, "/sys/stats", conf.API.SysStatURI)
	assert.Equal(t, "/metrics", conf.API.MetricURI)
	assert.Equal(t, "/healthz", conf.API.HealthURI)
}

func TestSanitizedCopy(t *testing.T) {
	cfg := &ConfYaml{}
	cfg.Core.CertBase64 = "cert-data"
	cfg.Core.KeyBase64 = "key-data"
	cfg.Core.HTTPProxy = "http://proxy:8080"
	cfg.Core.CertPath = "/path/to/cert.pem"
	cfg.Core.KeyPath = "/path/to/key.pem"
	cfg.Core.FeedbackHeader = []string{"x-api-key:secret123", "x-auth-key:secret456"}
	cfg.Android.KeyPath = "/path/to/key.json"
	cfg.Android.Credential = "fcm-credential"
	cfg.Huawei.AppSecret = "hms-secret"
	cfg.Ios.KeyPath = "/path/to/ios.p8"
	cfg.Ios.KeyBase64 = "ios-key-data"
	cfg.Ios.Password = "ios-password"
	cfg.Ios.KeyID = "ABCDE12345"
	cfg.Ios.TeamID = "TEAM123456"
	cfg.Queue.Redis.Username = "queue-user"
	cfg.Queue.Redis.Password = "queue-pass"
	cfg.Stat.Redis.Username = "stat-user"
	cfg.Stat.Redis.Password = "stat-pass"

	// Set a non-sensitive field to verify it's preserved
	cfg.Core.Port = "8088"

	sanitized := cfg.SanitizedCopy()

	// All sensitive fields should be redacted
	assert.Equal(t, "[REDACTED]", sanitized.Core.CertBase64)
	assert.Equal(t, "[REDACTED]", sanitized.Core.KeyBase64)
	assert.Equal(t, "[REDACTED]", sanitized.Core.HTTPProxy)
	assert.Equal(t, "[REDACTED]", sanitized.Core.CertPath)
	assert.Equal(t, "[REDACTED]", sanitized.Core.KeyPath)
	assert.Len(t, sanitized.Core.FeedbackHeader, 2)
	assert.Equal(t, "[REDACTED]", sanitized.Core.FeedbackHeader[0])
	assert.Equal(t, "[REDACTED]", sanitized.Core.FeedbackHeader[1])
	assert.Equal(t, "[REDACTED]", sanitized.Android.KeyPath)
	assert.Equal(t, "[REDACTED]", sanitized.Android.Credential)
	assert.Equal(t, "[REDACTED]", sanitized.Huawei.AppSecret)
	assert.Equal(t, "[REDACTED]", sanitized.Ios.KeyPath)
	assert.Equal(t, "[REDACTED]", sanitized.Ios.KeyBase64)
	assert.Equal(t, "[REDACTED]", sanitized.Ios.Password)
	assert.Equal(t, "[REDACTED]", sanitized.Ios.KeyID)
	assert.Equal(t, "[REDACTED]", sanitized.Ios.TeamID)
	assert.Equal(t, "[REDACTED]", sanitized.Queue.Redis.Username)
	assert.Equal(t, "[REDACTED]", sanitized.Queue.Redis.Password)
	assert.Equal(t, "[REDACTED]", sanitized.Stat.Redis.Username)
	assert.Equal(t, "[REDACTED]", sanitized.Stat.Redis.Password)

	// Non-sensitive fields should be preserved
	assert.Equal(t, "8088", sanitized.Core.Port)

	// Original config must NOT be modified
	assert.Equal(t, "/path/to/cert.pem", cfg.Core.CertPath)
	assert.Equal(t, "/path/to/key.pem", cfg.Core.KeyPath)
	assert.Equal(t, "x-api-key:secret123", cfg.Core.FeedbackHeader[0])
	assert.Equal(t, "cert-data", cfg.Core.CertBase64)
	assert.Equal(t, "fcm-credential", cfg.Android.Credential)
	assert.Equal(t, "ios-password", cfg.Ios.Password)
	assert.Equal(t, "stat-pass", cfg.Stat.Redis.Password)
}

func TestSanitizedCopyEmptyFields(t *testing.T) {
	cfg := &ConfYaml{}
	// All sensitive fields are empty by default

	sanitized := cfg.SanitizedCopy()

	// Empty fields should remain empty, not become "[REDACTED]"
	assert.Empty(t, sanitized.Core.CertBase64)
	assert.Empty(t, sanitized.Core.KeyBase64)
	assert.Empty(t, sanitized.Core.HTTPProxy)
	assert.Empty(t, sanitized.Core.CertPath)
	assert.Empty(t, sanitized.Core.KeyPath)
	assert.Nil(t, sanitized.Core.FeedbackHeader)
	assert.Empty(t, sanitized.Android.KeyPath)
	assert.Empty(t, sanitized.Android.Credential)
	assert.Empty(t, sanitized.Huawei.AppSecret)
	assert.Empty(t, sanitized.Ios.KeyPath)
	assert.Empty(t, sanitized.Ios.KeyBase64)
	assert.Empty(t, sanitized.Ios.Password)
	assert.Empty(t, sanitized.Ios.KeyID)
	assert.Empty(t, sanitized.Ios.TeamID)
	assert.Empty(t, sanitized.Queue.Redis.Username)
	assert.Empty(t, sanitized.Queue.Redis.Password)
	assert.Empty(t, sanitized.Stat.Redis.Username)
	assert.Empty(t, sanitized.Stat.Redis.Password)
}

// ---------- new validation tests ----------

func TestValidateHostPort(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
		errMsg  string
	}{
		{name: "empty", addr: "", wantErr: true, errMsg: "must not be empty"},
		{name: "valid ip:port", addr: "127.0.0.1:6379", wantErr: false},
		{name: "valid host:port", addr: "localhost:4150", wantErr: false},
		{name: "valid high port", addr: "0.0.0.0:65535", wantErr: false},
		{name: "missing port", addr: "127.0.0.1", wantErr: true, errMsg: "invalid host:port"},
		{name: "invalid port", addr: "127.0.0.1:abc", wantErr: true, errMsg: "invalid port"},
		{name: "port out of range", addr: "127.0.0.1:99999", wantErr: true, errMsg: "invalid port"},
		{name: "just colon", addr: ":", wantErr: true, errMsg: "host must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostPort(tt.addr)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		errMsg  string
	}{
		{name: "empty is valid", raw: "", wantErr: false},
		{name: "valid http", raw: "http://example.com/hook", wantErr: false},
		{name: "valid https", raw: "https://example.com/hook", wantErr: false},
		{name: "valid with port", raw: "http://localhost:8080/callback", wantErr: false},
		{name: "ftp scheme", raw: "ftp://example.com", wantErr: true, errMsg: "http or https"},
		{name: "no scheme", raw: "example.com", wantErr: true, errMsg: "http or https"},
		{name: "no host", raw: "http://", wantErr: true, errMsg: "must have a host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{name: "empty is valid", level: "", wantErr: false},
		{name: "debug", level: "debug", wantErr: false},
		{name: "info", level: "info", wantErr: false},
		{name: "warn", level: "warn", wantErr: false},
		{name: "error", level: "error", wantErr: false},
		{name: "fatal", level: "fatal", wantErr: false},
		{name: "trace", level: "trace", wantErr: false},
		{name: "panic", level: "panic", wantErr: false},
		{name: "disable", level: "disable", wantErr: false},
		{name: "case insensitive", level: "DEBUG", wantErr: false},
		{name: "invalid", level: "verbose", wantErr: true},
		{name: "garbage", level: "not-a-level", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogLevel(tt.level)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateStatEngine(t *testing.T) {
	tests := []struct {
		name    string
		engine  string
		stat    SectionStat
		wantErr bool
		errMsg  string
	}{
		{
			name:    "memory valid",
			engine:  "memory",
			stat:    SectionStat{},
			wantErr: false,
		},
		{
			name:    "empty engine",
			engine:  "",
			wantErr: true,
			errMsg:  "must not be empty",
		},
		{
			name:    "unsupported engine",
			engine:  "mongodb",
			wantErr: true,
			errMsg:  "unsupported stat engine",
		},
		{
			name:   "redis valid",
			engine: "redis",
			stat: SectionStat{
				Redis: SectionRedis{Addr: "localhost:6379"},
			},
			wantErr: false,
		},
		{
			name:    "redis empty addr",
			engine:  "redis",
			stat:    SectionStat{Redis: SectionRedis{Addr: ""}},
			wantErr: true,
			errMsg:  "redis addr",
		},
		{
			name:    "redis invalid addr",
			engine:  "redis",
			stat:    SectionStat{Redis: SectionRedis{Addr: "no-port"}},
			wantErr: true,
			errMsg:  "redis address",
		},
		{
			name:   "boltdb valid",
			engine: "boltdb",
			stat: SectionStat{
				BoltDB: SectionBoltDB{Path: "bolt.db", Bucket: "gorush"},
			},
			wantErr: false,
		},
		{
			name:    "boltdb empty path",
			engine:  "boltdb",
			stat:    SectionStat{BoltDB: SectionBoltDB{Path: "", Bucket: "gorush"}},
			wantErr: true,
			errMsg:  "boltdb path",
		},
		{
			name:    "boltdb empty bucket",
			engine:  "boltdb",
			stat:    SectionStat{BoltDB: SectionBoltDB{Path: "bolt.db", Bucket: ""}},
			wantErr: true,
			errMsg:  "boltdb bucket",
		},
		{
			name:    "buntdb valid",
			engine:  "buntdb",
			stat:    SectionStat{BuntDB: SectionBuntDB{Path: "bunt.db"}},
			wantErr: false,
		},
		{
			name:    "buntdb empty path",
			engine:  "buntdb",
			stat:    SectionStat{BuntDB: SectionBuntDB{Path: ""}},
			wantErr: true,
			errMsg:  "buntdb path",
		},
		{
			name:    "leveldb valid",
			engine:  "leveldb",
			stat:    SectionStat{LevelDB: SectionLevelDB{Path: "level.db"}},
			wantErr: false,
		},
		{
			name:    "leveldb empty path",
			engine:  "leveldb",
			stat:    SectionStat{LevelDB: SectionLevelDB{Path: ""}},
			wantErr: true,
			errMsg:  "leveldb path",
		},
		{
			name:    "badger valid",
			engine:  "badger",
			stat:    SectionStat{BadgerDB: SectionBadgerDB{Path: "badger.db"}},
			wantErr: false,
		},
		{
			name:    "badger empty path",
			engine:  "badger",
			stat:    SectionStat{BadgerDB: SectionBadgerDB{Path: ""}},
			wantErr: true,
			errMsg:  "badgerdb path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStatEngine(tt.engine, tt.stat)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateQueueEngine(t *testing.T) {
	tests := []struct {
		name    string
		engine  string
		q       SectionQueue
		wantErr bool
		errMsg  string
	}{
		{
			name:    "local valid",
			engine:  "local",
			q:       SectionQueue{},
			wantErr: false,
		},
		{
			name:    "empty engine",
			engine:  "",
			wantErr: true,
			errMsg:  "must not be empty",
		},
		{
			name:    "unsupported engine",
			engine:  "kafka",
			wantErr: true,
			errMsg:  "unsupported queue engine",
		},
		{
			name:   "nsq valid",
			engine: "nsq",
			q: SectionQueue{
				NSQ: SectionNSQ{Addr: "127.0.0.1:4150", Topic: "gorush", Channel: "gorush"},
			},
			wantErr: false,
		},
		{
			name:    "nsq empty addr",
			engine:  "nsq",
			q:       SectionQueue{NSQ: SectionNSQ{Addr: "", Topic: "t", Channel: "c"}},
			wantErr: true,
			errMsg:  "nsq addr",
		},
		{
			name:    "nsq invalid addr",
			engine:  "nsq",
			q:       SectionQueue{NSQ: SectionNSQ{Addr: "bad", Topic: "t", Channel: "c"}},
			wantErr: true,
			errMsg:  "nsq address",
		},
		{
			name:    "nsq empty topic",
			engine:  "nsq",
			q:       SectionQueue{NSQ: SectionNSQ{Addr: "127.0.0.1:4150", Topic: "", Channel: "c"}},
			wantErr: true,
			errMsg:  "nsq topic",
		},
		{
			name:    "nsq empty channel",
			engine:  "nsq",
			q:       SectionQueue{NSQ: SectionNSQ{Addr: "127.0.0.1:4150", Topic: "t", Channel: ""}},
			wantErr: true,
			errMsg:  "nsq channel",
		},
		{
			name:   "nats valid",
			engine: "nats",
			q: SectionQueue{
				NATS: SectionNATS{Addr: "127.0.0.1:4222", Subj: "gorush", Queue: "gorush"},
			},
			wantErr: false,
		},
		{
			name:    "nats empty addr",
			engine:  "nats",
			q:       SectionQueue{NATS: SectionNATS{Addr: "", Subj: "s", Queue: "q"}},
			wantErr: true,
			errMsg:  "nats addr",
		},
		{
			name:    "nats invalid addr",
			engine:  "nats",
			q:       SectionQueue{NATS: SectionNATS{Addr: "bad", Subj: "s", Queue: "q"}},
			wantErr: true,
			errMsg:  "nats address",
		},
		{
			name:    "nats empty subject",
			engine:  "nats",
			q:       SectionQueue{NATS: SectionNATS{Addr: "127.0.0.1:4222", Subj: "", Queue: "q"}},
			wantErr: true,
			errMsg:  "nats subject",
		},
		{
			name:    "nats empty queue",
			engine:  "nats",
			q:       SectionQueue{NATS: SectionNATS{Addr: "127.0.0.1:4222", Subj: "s", Queue: ""}},
			wantErr: true,
			errMsg:  "nats queue",
		},
		{
			name:   "redis valid",
			engine: "redis",
			q: SectionQueue{
				Redis: SectionRedisQueue{
					Addr: "127.0.0.1:6379", StreamName: "gorush",
					Group: "gorush", Consumer: "gorush",
				},
			},
			wantErr: false,
		},
		{
			name:    "redis empty addr",
			engine:  "redis",
			q:       SectionQueue{Redis: SectionRedisQueue{Addr: "", StreamName: "s", Group: "g", Consumer: "c"}},
			wantErr: true,
			errMsg:  "redis addr",
		},
		{
			name:    "redis invalid addr",
			engine:  "redis",
			q:       SectionQueue{Redis: SectionRedisQueue{Addr: "bad", StreamName: "s", Group: "g", Consumer: "c"}},
			wantErr: true,
			errMsg:  "redis address",
		},
		{
			name:    "redis empty stream",
			engine:  "redis",
			q:       SectionQueue{Redis: SectionRedisQueue{Addr: "127.0.0.1:6379", StreamName: "", Group: "g", Consumer: "c"}},
			wantErr: true,
			errMsg:  "stream_name",
		},
		{
			name:    "redis empty group",
			engine:  "redis",
			q:       SectionQueue{Redis: SectionRedisQueue{Addr: "127.0.0.1:6379", StreamName: "s", Group: "", Consumer: "c"}},
			wantErr: true,
			errMsg:  "group",
		},
		{
			name:    "redis empty consumer",
			engine:  "redis",
			q:       SectionQueue{Redis: SectionRedisQueue{Addr: "127.0.0.1:6379", StreamName: "s", Group: "g", Consumer: ""}},
			wantErr: true,
			errMsg:  "consumer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQueueEngine(tt.engine, tt.q)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		core    SectionCore
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no SSL no AutoTLS",
			core:    SectionCore{},
			wantErr: false,
		},
		{
			name: "SSL with file certs",
			core: SectionCore{
				SSL: true, CertPath: "cert.pem", KeyPath: "key.pem",
			},
			wantErr: false,
		},
		{
			name: "SSL with base64 certs",
			core: SectionCore{
				SSL: true, CertBase64: "Y2VydA==", KeyBase64: "a2V5",
			},
			wantErr: false,
		},
		{
			name:    "SSL enabled but no certs",
			core:    SectionCore{SSL: true},
			wantErr: true,
			errMsg:  "no certificate configured",
		},
		{
			name: "SSL with only cert_path",
			core: SectionCore{
				SSL: true, CertPath: "cert.pem",
			},
			wantErr: true,
			errMsg:  "cert_path and key_path must both be set",
		},
		{
			name: "SSL with only key_path",
			core: SectionCore{
				SSL: true, KeyPath: "key.pem",
			},
			wantErr: true,
			errMsg:  "cert_path and key_path must both be set",
		},
		{
			name: "SSL with only cert_base64",
			core: SectionCore{
				SSL: true, CertBase64: "Y2VydA==",
			},
			wantErr: true,
			errMsg:  "cert_base64 and key_base64 must both be set",
		},
		{
			name: "SSL with only key_base64",
			core: SectionCore{
				SSL: true, KeyBase64: "a2V5",
			},
			wantErr: true,
			errMsg:  "cert_base64 and key_base64 must both be set",
		},
		{
			name: "AutoTLS valid",
			core: SectionCore{
				AutoTLS: SectionAutoTLS{Enabled: true, Host: "example.com", Folder: ".cache"},
			},
			wantErr: false,
		},
		{
			name: "AutoTLS empty host",
			core: SectionCore{
				AutoTLS: SectionAutoTLS{Enabled: true, Host: "", Folder: ".cache"},
			},
			wantErr: true,
			errMsg:  "auto_tls host",
		},
		{
			name: "AutoTLS empty folder",
			core: SectionCore{
				AutoTLS: SectionAutoTLS{Enabled: true, Host: "example.com", Folder: ""},
			},
			wantErr: true,
			errMsg:  "auto_tls folder",
		},
		{
			name: "SSL with AutoTLS enabled skips cert check",
			core: SectionCore{
				SSL:     true,
				AutoTLS: SectionAutoTLS{Enabled: true, Host: "example.com", Folder: ".cache"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConfig(tt.core)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateGRPC(t *testing.T) {
	tests := []struct {
		name    string
		grpc    SectionGRPC
		wantErr bool
		errMsg  string
	}{
		{
			name:    "disabled no port",
			grpc:    SectionGRPC{Enabled: false},
			wantErr: false,
		},
		{
			name:    "enabled valid port",
			grpc:    SectionGRPC{Enabled: true, Port: "9000"},
			wantErr: false,
		},
		{
			name:    "enabled empty port",
			grpc:    SectionGRPC{Enabled: true, Port: ""},
			wantErr: true,
			errMsg:  "must not be empty",
		},
		{
			name:    "enabled invalid port",
			grpc:    SectionGRPC{Enabled: true, Port: "abc"},
			wantErr: true,
			errMsg:  "invalid grpc port",
		},
		{
			name:    "enabled port out of range",
			grpc:    SectionGRPC{Enabled: true, Port: "99999"},
			wantErr: true,
			errMsg:  "invalid grpc port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGRPC(tt.grpc)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// validMinimalConfig returns a minimal valid ConfYaml for testing.
func validMinimalConfig() *ConfYaml {
	return &ConfYaml{
		Core: SectionCore{
			Port:            "8088",
			Mode:            "release",
			WorkerNum:       4,
			QueueNum:        8192,
			MaxNotification: 100,
		},
		Log: SectionLog{
			AccessLevel: "debug",
			ErrorLevel:  "error",
		},
		Stat:  SectionStat{Engine: "memory"},
		Queue: SectionQueue{Engine: "local"},
	}
}

func TestValidateConfigComprehensive(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ConfYaml
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid minimal config",
			cfg:     validMinimalConfig(),
			wantErr: false,
		},
		{
			name: "invalid core port",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.Port = "99999"
				return c
			}(),
			wantErr: true,
			errMsg:  "invalid core port",
		},
		{
			name: "invalid mode",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.Mode = "production"
				return c
			}(),
			wantErr: true,
			errMsg:  "invalid core mode",
		},
		{
			name: "negative worker_num",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.WorkerNum = -1
				return c
			}(),
			wantErr: true,
			errMsg:  "worker_num must be non-negative",
		},
		{
			name: "negative queue_num",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.QueueNum = -1
				return c
			}(),
			wantErr: true,
			errMsg:  "queue_num must be non-negative",
		},
		{
			name: "negative max_notification",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.MaxNotification = -1
				return c
			}(),
			wantErr: true,
			errMsg:  "max_notification must be non-negative",
		},
		{
			name: "invalid log access_level",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Log.AccessLevel = "verbose"
				return c
			}(),
			wantErr: true,
			errMsg:  "invalid log access_level",
		},
		{
			name: "invalid log error_level",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Log.ErrorLevel = "nope"
				return c
			}(),
			wantErr: true,
			errMsg:  "invalid log error_level",
		},
		{
			name: "invalid http proxy",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.HTTPProxy = "ftp://bad-scheme"
				return c
			}(),
			wantErr: true,
			errMsg:  "invalid http_proxy",
		},
		{
			name: "valid http proxy",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.HTTPProxy = "http://proxy:8080"
				return c
			}(),
			wantErr: false,
		},
		{
			name: "invalid feedback URL",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.FeedbackURL = "ftp://invalid"
				return c
			}(),
			wantErr: true,
			errMsg:  "invalid feedback_hook_url",
		},
		{
			name: "valid feedback URL",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.FeedbackURL = "https://hooks.example.com/feedback"
				c.Core.FeedbackTimeout = 10
				return c
			}(),
			wantErr: false,
		},
		{
			name: "feedback URL with zero timeout",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.FeedbackURL = "https://hooks.example.com/feedback"
				c.Core.FeedbackTimeout = 0
				return c
			}(),
			wantErr: true,
			errMsg:  "feedback_timeout must be positive",
		},
		{
			name: "invalid stat engine",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Stat.Engine = "mongodb"
				return c
			}(),
			wantErr: true,
			errMsg:  "stat engine",
		},
		{
			name: "invalid queue engine",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Queue.Engine = "kafka"
				return c
			}(),
			wantErr: true,
			errMsg:  "queue engine",
		},
		{
			name: "grpc enabled invalid port",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.GRPC = SectionGRPC{Enabled: true, Port: "bad"}
				return c
			}(),
			wantErr: true,
			errMsg:  "invalid grpc port",
		},
		{
			name: "grpc enabled valid",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.GRPC = SectionGRPC{Enabled: true, Port: "9000"}
				return c
			}(),
			wantErr: false,
		},
		{
			name: "port conflict HTTP and gRPC",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.Port = "9000"
				c.Core.Enabled = true
				c.GRPC = SectionGRPC{Enabled: true, Port: "9000"}
				return c
			}(),
			wantErr: true,
			errMsg:  "must not be the same",
		},
		{
			name: "SSL without certs",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.SSL = true
				return c
			}(),
			wantErr: true,
			errMsg:  "no certificate configured",
		},
		{
			name: "SSL with certs valid",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.SSL = true
				c.Core.CertPath = "cert.pem"
				c.Core.KeyPath = "key.pem"
				return c
			}(),
			wantErr: false,
		},
		{
			name: "AutoTLS empty host",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Core.AutoTLS = SectionAutoTLS{Enabled: true, Folder: ".cache"}
				return c
			}(),
			wantErr: true,
			errMsg:  "auto_tls host",
		},
		{
			name: "queue nsq invalid addr",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Queue.Engine = "nsq"
				c.Queue.NSQ = SectionNSQ{Addr: "bad", Topic: "t", Channel: "c"}
				return c
			}(),
			wantErr: true,
			errMsg:  "queue engine",
		},
		{
			name: "queue redis valid",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Queue.Engine = "redis"
				c.Queue.Redis = SectionRedisQueue{
					Addr: "127.0.0.1:6379", StreamName: "gorush",
					Group: "gorush", Consumer: "gorush",
				}
				return c
			}(),
			wantErr: false,
		},
		{
			name: "stat redis invalid addr",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Stat.Engine = "redis"
				c.Stat.Redis = SectionRedis{Addr: "no-port"}
				return c
			}(),
			wantErr: true,
			errMsg:  "stat engine",
		},
		{
			name: "stat boltdb valid",
			cfg: func() *ConfYaml {
				c := validMinimalConfig()
				c.Stat.Engine = "boltdb"
				c.Stat.BoltDB = SectionBoltDB{Path: "bolt.db", Bucket: "gorush"}
				return c
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateConfigWithDefaultConfig verifies the default config passes validation.
func TestValidateConfigWithDefaultConfig(t *testing.T) {
	cfg, err := LoadConf()
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
}

// TestValidateConfigWithTestdataConfig verifies the testdata config passes validation.
func TestValidateConfigWithTestdataConfig(t *testing.T) {
	cfg, err := LoadConf("testdata/config.yml")
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
}

// TestValidateConfigWithRedisDBConfig verifies the redis_db_config testdata passes validation.
func TestValidateConfigWithRedisDBConfig(t *testing.T) {
	cfg, err := LoadConf("testdata/redis_db_config.yml")
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
}
