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
					Port:    "8088",
					Address: "0.0.0.0",
				},
				Stat: SectionStat{
					Engine: "memory",
				},
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

// checkValidationErr asserts the error returned by a validator matches the
// expectation of the table-driven test case.
func checkValidationErr(t *testing.T, err error, wantErr bool, errMsg string) {
	t.Helper()
	if wantErr {
		require.Error(t, err)
		if errMsg != "" {
			assert.Contains(t, err.Error(), errMsg)
		}
		return
	}
	require.NoError(t, err)
}

func TestValidateGRPCConfig(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
		errMsg  string
	}{
		{name: "empty port uses default", port: "", wantErr: false},
		{name: "valid port", port: "9000", wantErr: false},
		{name: "out of range port", port: "99999", wantErr: true, errMsg: "invalid gRPC port"},
		{name: "non-numeric port", port: "abc", wantErr: true, errMsg: "invalid gRPC port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConfYaml{}
			cfg.GRPC.Port = tt.port
			checkValidationErr(t, ValidateGRPCConfig(cfg), tt.wantErr, tt.errMsg)
		})
	}
}

func TestValidateQueueConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*ConfYaml)
		wantErr bool
		errMsg  string
	}{
		{name: "empty engine uses default", setup: func(c *ConfYaml) {}, wantErr: false},
		{
			name:    "local engine needs no address",
			setup:   func(c *ConfYaml) { c.Queue.Engine = "local" },
			wantErr: false,
		},
		{
			name:    "unknown engine",
			setup:   func(c *ConfYaml) { c.Queue.Engine = "kafka" },
			wantErr: true,
			errMsg:  "invalid queue engine",
		},
		{
			name: "valid nsq address",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "nsq"
				c.Queue.NSQ.Addr = "127.0.0.1:4150"
			},
			wantErr: false,
		},
		{
			name: "nsq address missing port",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "nsq"
				c.Queue.NSQ.Addr = "127.0.0.1"
			},
			wantErr: true,
			errMsg:  "invalid NSQ address",
		},
		{
			name: "nsq address out of range port",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "nsq"
				c.Queue.NSQ.Addr = "127.0.0.1:99999"
			},
			wantErr: true,
			errMsg:  "invalid NSQ address",
		},
		{
			name: "nats address with scheme",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "nats"
				c.Queue.NATS.Addr = "nats://127.0.0.1:4222"
			},
			wantErr: false,
		},
		{
			name: "nats multi-node cluster",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "nats"
				c.Queue.NATS.Addr = "127.0.0.1:4222,127.0.0.1:4223"
			},
			wantErr: false,
		},
		{
			name: "nats empty address",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "nats"
				c.Queue.NATS.Addr = ""
			},
			wantErr: true,
			errMsg:  "invalid NATS address",
		},
		{
			name: "valid redis queue address",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "redis"
				c.Queue.Redis.Addr = "127.0.0.1:6379"
			},
			wantErr: false,
		},
		{
			name: "redis queue address missing port",
			setup: func(c *ConfYaml) {
				c.Queue.Engine = "redis"
				c.Queue.Redis.Addr = "127.0.0.1"
			},
			wantErr: true,
			errMsg:  "invalid Redis queue address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConfYaml{}
			tt.setup(cfg)
			checkValidationErr(t, ValidateQueueConfig(cfg), tt.wantErr, tt.errMsg)
		})
	}
}

func TestValidateFeedbackConfig(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		timeout int64
		wantErr bool
		errMsg  string
	}{
		{name: "empty url skips validation", url: "", timeout: 0, wantErr: false},
		{name: "valid http url", url: "http://example.com/hook", timeout: 10, wantErr: false},
		{name: "valid https url", url: "https://example.com/hook", timeout: 5, wantErr: false},
		{
			name:    "missing scheme",
			url:     "example.com/hook",
			timeout: 10,
			wantErr: true,
			errMsg:  "scheme",
		},
		{
			name:    "unsupported scheme",
			url:     "ftp://example.com/hook",
			timeout: 10,
			wantErr: true,
			errMsg:  "scheme",
		},
		{
			name:    "missing host",
			url:     "http://",
			timeout: 10,
			wantErr: true,
			errMsg:  "host",
		},
		{
			name:    "zero timeout",
			url:     "http://example.com/hook",
			timeout: 0,
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name:    "negative timeout",
			url:     "http://example.com/hook",
			timeout: -1,
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConfYaml{}
			cfg.Core.FeedbackURL = tt.url
			cfg.Core.FeedbackTimeout = tt.timeout
			checkValidationErr(t, ValidateFeedbackConfig(cfg), tt.wantErr, tt.errMsg)
		})
	}
}

func TestValidateTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*ConfYaml)
		wantErr bool
		errMsg  string
	}{
		{name: "no tls", setup: func(c *ConfYaml) {}, wantErr: false},
		{
			name: "ssl with file cert pair",
			setup: func(c *ConfYaml) {
				c.Core.SSL = true
				c.Core.CertPath = "cert.pem"
				c.Core.KeyPath = "key.pem"
			},
			wantErr: false,
		},
		{
			name: "ssl with base64 cert pair",
			setup: func(c *ConfYaml) {
				c.Core.SSL = true
				c.Core.CertBase64 = "Y2VydA=="
				c.Core.KeyBase64 = "a2V5"
			},
			wantErr: false,
		},
		{
			name: "ssl with only cert path",
			setup: func(c *ConfYaml) {
				c.Core.SSL = true
				c.Core.CertPath = "cert.pem"
			},
			wantErr: true,
			errMsg:  "certificate/key pair",
		},
		{
			name: "ssl without any cert",
			setup: func(c *ConfYaml) {
				c.Core.SSL = true
			},
			wantErr: true,
			errMsg:  "certificate/key pair",
		},
		{
			name: "auto tls with host",
			setup: func(c *ConfYaml) {
				c.Core.AutoTLS.Enabled = true
				c.Core.AutoTLS.Host = "example.com"
			},
			wantErr: false,
		},
		{
			name: "auto tls without host",
			setup: func(c *ConfYaml) {
				c.Core.AutoTLS.Enabled = true
			},
			wantErr: true,
			errMsg:  "auto_tls.host",
		},
		{
			name: "auto tls without host takes precedence over ssl",
			setup: func(c *ConfYaml) {
				c.Core.AutoTLS.Enabled = true
				c.Core.SSL = true
				c.Core.CertPath = "cert.pem"
				c.Core.KeyPath = "key.pem"
			},
			wantErr: true,
			errMsg:  "auto_tls.host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConfYaml{}
			tt.setup(cfg)
			checkValidationErr(t, ValidateTLSConfig(cfg), tt.wantErr, tt.errMsg)
		})
	}
}

func TestValidateStatConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*ConfYaml)
		wantErr bool
		errMsg  string
	}{
		{name: "empty engine uses default", setup: func(c *ConfYaml) {}, wantErr: false},
		{name: "memory engine", setup: func(c *ConfYaml) { c.Stat.Engine = "memory" }, wantErr: false},
		{name: "boltdb engine", setup: func(c *ConfYaml) { c.Stat.Engine = "boltdb" }, wantErr: false},
		{
			name:    "unknown engine",
			setup:   func(c *ConfYaml) { c.Stat.Engine = "mysql" },
			wantErr: true,
			errMsg:  "invalid stat engine",
		},
		{
			name: "redis engine valid address",
			setup: func(c *ConfYaml) {
				c.Stat.Engine = "redis"
				c.Stat.Redis.Addr = "localhost:6379"
			},
			wantErr: false,
		},
		{
			name: "redis engine empty address skips check",
			setup: func(c *ConfYaml) {
				c.Stat.Engine = "redis"
				c.Stat.Redis.Addr = ""
			},
			wantErr: false,
		},
		{
			name: "redis engine address missing port",
			setup: func(c *ConfYaml) {
				c.Stat.Engine = "redis"
				c.Stat.Redis.Addr = "localhost"
			},
			wantErr: true,
			errMsg:  "invalid Redis address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConfYaml{}
			tt.setup(cfg)
			checkValidationErr(t, ValidateStatConfig(cfg), tt.wantErr, tt.errMsg)
		})
	}
}

// TestValidateConfigSubsystems verifies that ValidateConfig aggregates every
// subsystem validator, so a single misconfiguration in any subsystem is caught
// up front.
func TestValidateConfigSubsystems(t *testing.T) {
	validConfig := func() *ConfYaml {
		cfg := &ConfYaml{}
		cfg.Core.Port = "8088"
		cfg.Core.Address = "0.0.0.0"
		cfg.GRPC.Port = "9000"
		cfg.Queue.Engine = "redis"
		cfg.Queue.Redis.Addr = "127.0.0.1:6379"
		cfg.Core.FeedbackURL = "https://example.com/hook"
		cfg.Core.FeedbackTimeout = 10
		cfg.Stat.Engine = "memory"
		return cfg
	}

	tests := []struct {
		name    string
		mutate  func(*ConfYaml)
		wantErr bool
		errMsg  string
	}{
		{name: "fully valid config", mutate: func(c *ConfYaml) {}, wantErr: false},
		{
			name:    "invalid grpc port",
			mutate:  func(c *ConfYaml) { c.GRPC.Port = "99999" },
			wantErr: true,
			errMsg:  "invalid gRPC port",
		},
		{
			name: "ssl without certs",
			mutate: func(c *ConfYaml) {
				c.Core.SSL = true
			},
			wantErr: true,
			errMsg:  "certificate/key pair",
		},
		{
			name:    "auto tls without host",
			mutate:  func(c *ConfYaml) { c.Core.AutoTLS.Enabled = true },
			wantErr: true,
			errMsg:  "auto_tls.host",
		},
		{
			name: "invalid queue address",
			mutate: func(c *ConfYaml) {
				c.Queue.Engine = "nsq"
				c.Queue.NSQ.Addr = "missing-port"
			},
			wantErr: true,
			errMsg:  "invalid NSQ address",
		},
		{
			name:    "invalid feedback url",
			mutate:  func(c *ConfYaml) { c.Core.FeedbackURL = "not-a-url" },
			wantErr: true,
			errMsg:  "scheme",
		},
		{
			name:    "non-positive feedback timeout",
			mutate:  func(c *ConfYaml) { c.Core.FeedbackTimeout = 0 },
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name:    "invalid stat engine",
			mutate:  func(c *ConfYaml) { c.Stat.Engine = "mongo" },
			wantErr: true,
			errMsg:  "invalid stat engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			checkValidationErr(t, ValidateConfig(cfg), tt.wantErr, tt.errMsg)
		})
	}
}

// TestValidateConfigDefaults guards the critical invariant that the default
// configuration and the reference testdata config both pass full validation.
func TestValidateConfigDefaults(t *testing.T) {
	defaults, err := LoadConf()
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(defaults), "default config must pass validation")

	fromFile, err := LoadConf("testdata/config.yml")
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(fromFile), "reference config must pass validation")
}
