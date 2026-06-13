package config

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// ConfYaml is config structure.
type ConfYaml struct {
	Core    SectionCore    `yaml:"core"`
	API     SectionAPI     `yaml:"api"`
	Android SectionAndroid `yaml:"android"`
	Huawei  SectionHuawei  `yaml:"huawei"`
	Ios     SectionIos     `yaml:"ios"`
	Queue   SectionQueue   `yaml:"queue"`
	Log     SectionLog     `yaml:"log"`
	Stat    SectionStat    `yaml:"stat"`
	GRPC    SectionGRPC    `yaml:"grpc"`
}

// SectionCore is sub section of config.
type SectionCore struct {
	Enabled         bool           `yaml:"enabled"`
	Address         string         `yaml:"address"`
	ShutdownTimeout int64          `yaml:"shutdown_timeout"`
	Port            string         `yaml:"port"`
	MaxNotification int64          `yaml:"max_notification"`
	WorkerNum       int64          `yaml:"worker_num"`
	QueueNum        int64          `yaml:"queue_num"`
	Mode            string         `yaml:"mode"`
	Sync            bool           `yaml:"sync"`
	SSL             bool           `yaml:"ssl"`
	CertPath        string         `yaml:"cert_path"`
	KeyPath         string         `yaml:"key_path"`
	CertBase64      string         `yaml:"cert_base64"`
	KeyBase64       string         `yaml:"key_base64"`
	HTTPProxy       string         `yaml:"http_proxy"`
	PID             SectionPID     `yaml:"pid"`
	AutoTLS         SectionAutoTLS `yaml:"auto_tls"`

	FeedbackURL     string   `yaml:"feedback_hook_url"`
	FeedbackTimeout int64    `yaml:"feedback_timeout"`
	FeedbackHeader  []string `yaml:"feedback_header"`
}

// SectionAutoTLS support Let's Encrypt setting.
type SectionAutoTLS struct {
	Enabled bool   `yaml:"enabled"`
	Folder  string `yaml:"folder"`
	Host    string `yaml:"host"`
}

// SectionAPI is sub section of config.
type SectionAPI struct {
	PushURI    string `yaml:"push_uri"`
	StatGoURI  string `yaml:"stat_go_uri"`
	StatAppURI string `yaml:"stat_app_uri"`
	ConfigURI  string `yaml:"config_uri"`
	SysStatURI string `yaml:"sys_stat_uri"`
	MetricURI  string `yaml:"metric_uri"`
	HealthURI  string `yaml:"health_uri"`
}

// SectionAndroid is sub section of config.
type SectionAndroid struct {
	Enabled    bool   `yaml:"enabled"`
	KeyPath    string `yaml:"key_path"`
	Credential string `yaml:"credential"`
	MaxRetry   int    `yaml:"max_retry"`
}

// SectionHuawei is sub section of config.
type SectionHuawei struct {
	Enabled   bool   `yaml:"enabled"`
	AppSecret string `yaml:"appsecret"`
	AppID     string `yaml:"appid"`
	MaxRetry  int    `yaml:"max_retry"`
}

// SectionIos is sub section of config.
type SectionIos struct {
	Enabled             bool   `yaml:"enabled"`
	KeyPath             string `yaml:"key_path"`
	KeyBase64           string `yaml:"key_base64"`
	KeyType             string `yaml:"key_type"`
	Password            string `yaml:"password"`
	Production          bool   `yaml:"production"`
	MaxConcurrentPushes uint   `yaml:"max_concurrent_pushes"`
	MaxRetry            int    `yaml:"max_retry"`
	KeyID               string `yaml:"key_id"`
	TeamID              string `yaml:"team_id"`
}

// SectionLog is sub section of config.
type SectionLog struct {
	Format       string `yaml:"format"`
	AccessLog    string `yaml:"access_log"`
	AccessLevel  string `yaml:"access_level"`
	ErrorLog     string `yaml:"error_log"`
	ErrorLevel   string `yaml:"error_level"`
	HideToken    bool   `yaml:"hide_token"`
	HideMessages bool   `yaml:"hide_messages"`
}

// SectionStat is sub section of config.
type SectionStat struct {
	Engine   string          `yaml:"engine"`
	Redis    SectionRedis    `yaml:"redis"`
	BoltDB   SectionBoltDB   `yaml:"boltdb"`
	BuntDB   SectionBuntDB   `yaml:"buntdb"`
	LevelDB  SectionLevelDB  `yaml:"leveldb"`
	BadgerDB SectionBadgerDB `yaml:"badgerdb"`
}

// SectionQueue is sub section of config.
type SectionQueue struct {
	Engine string            `yaml:"engine"`
	NSQ    SectionNSQ        `yaml:"nsq"`
	NATS   SectionNATS       `yaml:"nats"`
	Redis  SectionRedisQueue `yaml:"redis"`
}

// SectionNSQ is sub section of config.
type SectionNSQ struct {
	Addr    string `yaml:"addr"`
	Topic   string `yaml:"topic"`
	Channel string `yaml:"channel"`
}

// SectionNATS is sub section of config.
type SectionNATS struct {
	Addr  string `yaml:"addr"`
	Subj  string `yaml:"subj"`
	Queue string `yaml:"queue"`
}

// SectionRedisQueue is sub section of config.
type SectionRedisQueue struct {
	Addr       string `yaml:"addr"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	DB         int    `yaml:"db"`
	StreamName string `yaml:"stream_name"`
	Group      string `yaml:"group"`
	Consumer   string `yaml:"consumer"`
	WithTLS    bool   `yaml:"with_tls"`
}

// SectionRedis is sub section of config.
type SectionRedis struct {
	Cluster  bool   `yaml:"cluster"`
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// SectionBoltDB is sub section of config.
type SectionBoltDB struct {
	Path   string `yaml:"path"`
	Bucket string `yaml:"bucket"`
}

// SectionBuntDB is sub section of config.
type SectionBuntDB struct {
	Path string `yaml:"path"`
}

// SectionLevelDB is sub section of config.
type SectionLevelDB struct {
	Path string `yaml:"path"`
}

// SectionBadgerDB is sub section of config.
type SectionBadgerDB struct {
	Path string `yaml:"path"`
}

// SectionPID is sub section of config.
type SectionPID struct {
	Enabled  bool   `yaml:"enabled"`
	Path     string `yaml:"path"`
	Override bool   `yaml:"override"`
}

// SectionGRPC is sub section of config.
type SectionGRPC struct {
	Enabled bool   `yaml:"enabled"`
	Port    string `yaml:"port"`
}

func setDefaults(v *viper.Viper) {
	// Core
	v.SetDefault("core.enabled", true)
	v.SetDefault("core.address", "")
	v.SetDefault("core.shutdown_timeout", 30)
	v.SetDefault("core.port", "8088")
	v.SetDefault("core.worker_num", 0)
	v.SetDefault("core.queue_num", 0)
	v.SetDefault("core.max_notification", 100)
	v.SetDefault("core.sync", false)
	v.SetDefault("core.feedback_hook_url", "")
	v.SetDefault("core.feedback_timeout", 10)
	v.SetDefault("core.mode", "release")
	v.SetDefault("core.ssl", false)
	v.SetDefault("core.cert_path", "cert.pem")
	v.SetDefault("core.key_path", "key.pem")
	v.SetDefault("core.cert_base64", "")
	v.SetDefault("core.key_base64", "")
	v.SetDefault("core.http_proxy", "")
	v.SetDefault("core.pid.enabled", false)
	v.SetDefault("core.pid.path", "gorush.pid")
	v.SetDefault("core.pid.override", true)
	v.SetDefault("core.auto_tls.enabled", false)
	v.SetDefault("core.auto_tls.folder", ".cache")
	v.SetDefault("core.auto_tls.host", "")

	// API
	v.SetDefault("api.push_uri", "/api/push")
	v.SetDefault("api.stat_go_uri", "/api/stat/go")
	v.SetDefault("api.stat_app_uri", "/api/stat/app")
	v.SetDefault("api.config_uri", "/api/config")
	v.SetDefault("api.sys_stat_uri", "/sys/stats")
	v.SetDefault("api.metric_uri", "/metrics")
	v.SetDefault("api.health_uri", "/healthz")

	// Android
	v.SetDefault("android.enabled", true)
	v.SetDefault("android.max_retry", 0)
	v.SetDefault("android.key_path", "")
	v.SetDefault("android.credential", "")

	// Huawei
	v.SetDefault("huawei.enabled", false)
	v.SetDefault("huawei.appsecret", "")
	v.SetDefault("huawei.appid", "")
	v.SetDefault("huawei.max_retry", 0)

	// iOS
	v.SetDefault("ios.enabled", false)
	v.SetDefault("ios.key_path", "")
	v.SetDefault("ios.key_base64", "")
	v.SetDefault("ios.key_type", "pem")
	v.SetDefault("ios.password", "")
	v.SetDefault("ios.production", false)
	v.SetDefault("ios.max_concurrent_pushes", uint(100))
	v.SetDefault("ios.max_retry", 0)
	v.SetDefault("ios.key_id", "")
	v.SetDefault("ios.team_id", "")

	// gRPC
	v.SetDefault("grpc.enabled", false)
	v.SetDefault("grpc.port", "9000")

	// Queue
	v.SetDefault("queue.engine", "local")
	v.SetDefault("queue.nsq.addr", "127.0.0.1:4150")
	v.SetDefault("queue.nsq.topic", "gorush")
	v.SetDefault("queue.nsq.channel", "gorush")
	v.SetDefault("queue.nats.addr", "127.0.0.1:4222")
	v.SetDefault("queue.nats.subj", "gorush")
	v.SetDefault("queue.nats.queue", "gorush")
	v.SetDefault("queue.redis.addr", "127.0.0.1:6379")
	v.SetDefault("queue.redis.group", "gorush")
	v.SetDefault("queue.redis.consumer", "gorush")
	v.SetDefault("queue.redis.stream_name", "gorush")
	v.SetDefault("queue.redis.with_tls", false)
	v.SetDefault("queue.redis.username", "")
	v.SetDefault("queue.redis.password", "")
	v.SetDefault("queue.redis.db", 0)

	// Stat
	v.SetDefault("stat.engine", "memory")
	v.SetDefault("stat.redis.cluster", false)
	v.SetDefault("stat.redis.addr", "localhost:6379")
	v.SetDefault("stat.redis.username", "")
	v.SetDefault("stat.redis.password", "")
	v.SetDefault("stat.redis.db", 0)
	v.SetDefault("stat.boltdb.path", "bolt.db")
	v.SetDefault("stat.boltdb.bucket", "gorush")
	v.SetDefault("stat.buntdb.path", "bunt.db")
	v.SetDefault("stat.leveldb.path", "level.db")
	v.SetDefault("stat.badgerdb.path", "badger.db")

	// Log
	v.SetDefault("log.format", "string")
	v.SetDefault("log.access_log", "stdout")
	v.SetDefault("log.access_level", "debug")
	v.SetDefault("log.error_log", "stderr")
	v.SetDefault("log.error_level", "error")
	v.SetDefault("log.hide_token", true)
	v.SetDefault("log.hide_messages", false)
}

// LoadConf load config from file and read in environment variables that match
func LoadConf(confPath ...string) (*ConfYaml, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigType("yaml")
	v.AutomaticEnv()         // read in environment variables that match
	v.SetEnvPrefix("gorush") // will be uppercased automatically
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If config path is provided, load from file
	if len(confPath) > 0 && confPath[0] != "" {
		content, err := os.ReadFile(confPath[0])
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", confPath[0], err)
		}

		if err := v.ReadConfig(bytes.NewBuffer(content)); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", confPath[0], err)
		}

		return loadConfigFromViper(v)
	}

	// Search for config.{yaml,yml,json,...} in standard directories.
	v.AddConfigPath("/etc/gorush/")
	v.AddConfigPath("$HOME/.gorush")
	v.AddConfigPath(".")
	v.SetConfigName("config")

	// If a config file is found, read it in.
	if err := v.ReadInConfig(); err != nil {
		// Only ignore "config file not found" — fail on parse/permission errors
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	} else {
		log.Println("Using config file:", v.ConfigFileUsed())
	}

	return loadConfigFromViper(v)
}

// loadConfigFromViper unmarshals viper config into the ConfYaml struct
func loadConfigFromViper(v *viper.Viper) (*ConfYaml, error) {
	conf := &ConfYaml{}

	err := v.Unmarshal(conf, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// GetStringSlice handles space-delimited env vars (e.g., "key1:val1 key2:val2")
	// which Unmarshal does not split automatically
	conf.Core.FeedbackHeader = v.GetStringSlice("core.feedback_header")

	if conf.Core.WorkerNum == 0 {
		conf.Core.WorkerNum = int64(runtime.NumCPU())
	}

	if conf.Core.QueueNum == 0 {
		conf.Core.QueueNum = 8192
	}

	return conf, nil
}

// ValidatePort validates that a port string is within valid range
func ValidatePort(port string) error {
	if port == "" {
		return nil // empty port is allowed, will use default
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port format: %s", port)
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port out of range (1-65535): %d", p)
	}
	return nil
}

// ValidateAddress validates that an address is not empty and contains valid characters
func ValidateAddress(addr string) error {
	if addr == "" {
		return nil // empty address is allowed, will bind to all interfaces
	}
	// Basic validation - check if it's a valid IP or hostname format
	if net.ParseIP(addr) == nil {
		// If not a valid IP, check if it could be a hostname (basic check)
		if len(addr) > 253 || strings.Contains(addr, "..") {
			return fmt.Errorf("invalid address format: %s", addr)
		}
	}
	return nil
}

// ValidatePIDPath validates and sanitizes PID file path to prevent path traversal
func ValidatePIDPath(pidPath string) error {
	if pidPath == "" {
		return nil
	}

	// Clean the path to resolve any . or .. elements
	cleanPath := filepath.Clean(pidPath)

	// Check for path traversal attempts by looking for ".." in the cleaned path
	// If a relative path attempts to traverse upwards, it will still have a ".." prefix after cleaning
	if !filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path traversal detected in PID path: %s", pidPath)
	}

	// Ensure the path is not trying to write to sensitive system directories
	sensitive := []string{"/etc/", "/usr/", "/var/", "/sys/", "/proc/"}
	for _, prefix := range sensitive {
		if strings.HasPrefix(cleanPath, prefix) {
			return fmt.Errorf("cannot write PID file to sensitive directory: %s", cleanPath)
		}
	}

	return nil
}

// validEngine reports whether value is one of the allowed engine names.
func validEngine(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

// validateHostPort validates a "host:port" address: the host must be a valid
// IP/hostname and the port must be a valid, non-empty port number.
func validateHostPort(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %s", addr)
	}
	if err := ValidateAddress(host); err != nil {
		return err
	}
	if port == "" {
		return fmt.Errorf("missing port in address: %s", addr)
	}
	return ValidatePort(port)
}

// validateQueueAddr validates a queue backend address. It accepts an optional
// "scheme://" prefix (e.g. nats://) and a comma-separated list of host:port
// entries (e.g. a multi-node NATS cluster), validating each entry.
func validateQueueAddr(addr string) error {
	if addr == "" {
		return errors.New("address can't be empty")
	}
	for _, entry := range strings.Split(addr, ",") {
		entry = strings.TrimSpace(entry)
		// Strip an optional scheme:// prefix before host:port validation.
		if i := strings.Index(entry, "://"); i != -1 {
			entry = entry[i+len("://"):]
		}
		if err := validateHostPort(entry); err != nil {
			return err
		}
	}
	return nil
}

// ValidateGRPCConfig validates the gRPC server settings consumed by
// rpc.RunGRPCServer.
func ValidateGRPCConfig(cfg *ConfYaml) error {
	if err := ValidatePort(cfg.GRPC.Port); err != nil {
		return fmt.Errorf("invalid gRPC port: %w", err)
	}
	return nil
}

// ValidateQueueConfig validates the queue engine and its backend address
// consumed by app.NewQueueWorker. An empty engine is treated as unset (the
// default engine is applied later).
func ValidateQueueConfig(cfg *ConfYaml) error {
	engine := cfg.Queue.Engine
	if engine == "" {
		return nil
	}
	if !validEngine(engine, "local", "nsq", "nats", "redis") {
		return fmt.Errorf("invalid queue engine: %s", engine)
	}

	switch engine {
	case "nsq":
		if err := validateQueueAddr(cfg.Queue.NSQ.Addr); err != nil {
			return fmt.Errorf("invalid NSQ address: %w", err)
		}
	case "nats":
		if err := validateQueueAddr(cfg.Queue.NATS.Addr); err != nil {
			return fmt.Errorf("invalid NATS address: %w", err)
		}
	case "redis":
		if err := validateQueueAddr(cfg.Queue.Redis.Addr); err != nil {
			return fmt.Errorf("invalid Redis queue address: %w", err)
		}
	}
	return nil
}

// ValidateFeedbackConfig validates the feedback webhook settings consumed by
// notify.DispatchFeedback. Validation only runs when a feedback URL is set.
func ValidateFeedbackConfig(cfg *ConfYaml) error {
	if cfg.Core.FeedbackURL == "" {
		return nil
	}

	u, err := url.Parse(cfg.Core.FeedbackURL)
	if err != nil {
		return fmt.Errorf("invalid feedback hook URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf(
			"feedback hook URL must use http or https scheme: %s",
			cfg.Core.FeedbackURL,
		)
	}
	if u.Host == "" {
		return fmt.Errorf("feedback hook URL must include a host: %s", cfg.Core.FeedbackURL)
	}
	if cfg.Core.FeedbackTimeout <= 0 {
		return fmt.Errorf(
			"feedback timeout must be positive, got %d",
			cfg.Core.FeedbackTimeout,
		)
	}
	return nil
}

// ValidateTLSConfig validates the TLS and AutoTLS settings consumed by the HTTP
// and gRPC servers, so that an SSL or AutoTLS misconfiguration fails at startup
// rather than when the server begins listening.
func ValidateTLSConfig(cfg *ConfYaml) error {
	if cfg.Core.AutoTLS.Enabled && cfg.Core.AutoTLS.Host == "" {
		return errors.New("auto_tls is enabled but auto_tls.host is empty")
	}

	if cfg.Core.SSL {
		hasFilePair := cfg.Core.CertPath != "" && cfg.Core.KeyPath != ""
		hasBase64Pair := cfg.Core.CertBase64 != "" && cfg.Core.KeyBase64 != ""
		if !hasFilePair && !hasBase64Pair {
			return errors.New(
				"ssl is enabled but no certificate/key pair " +
					"(cert_path+key_path or cert_base64+key_base64) is configured",
			)
		}
	}
	return nil
}

// ValidateStatConfig validates the stat storage engine consumed by
// status.InitAppStatus. An empty engine is treated as unset (the default engine
// is applied later).
func ValidateStatConfig(cfg *ConfYaml) error {
	engine := cfg.Stat.Engine
	if engine == "" {
		return nil
	}
	if !validEngine(engine, "memory", "redis", "boltdb", "buntdb", "leveldb", "badger") {
		return fmt.Errorf("invalid stat engine: %s", engine)
	}

	if engine == "redis" && cfg.Stat.Redis.Addr != "" {
		if err := validateHostPort(cfg.Stat.Redis.Addr); err != nil {
			return fmt.Errorf("invalid Redis address: %w", err)
		}
	}
	return nil
}

// ValidateConfig validates the full configuration before startup so that bad
// values are rejected up front in a single pass, rather than failing later
// inside the HTTP, gRPC, queue, or feedback subsystems once they start.
func ValidateConfig(cfg *ConfYaml) error {
	if err := ValidatePort(cfg.Core.Port); err != nil {
		return fmt.Errorf("invalid core port: %w", err)
	}

	if err := ValidateAddress(cfg.Core.Address); err != nil {
		return fmt.Errorf("invalid core address: %w", err)
	}

	if err := ValidatePIDPath(cfg.Core.PID.Path); err != nil {
		return fmt.Errorf("invalid PID path: %w", err)
	}

	if err := ValidateGRPCConfig(cfg); err != nil {
		return err
	}

	if err := ValidateTLSConfig(cfg); err != nil {
		return err
	}

	if err := ValidateQueueConfig(cfg); err != nil {
		return err
	}

	if err := ValidateFeedbackConfig(cfg); err != nil {
		return err
	}

	if err := ValidateStatConfig(cfg); err != nil {
		return err
	}

	return nil
}

// SanitizedCopy returns a copy of the config with sensitive fields redacted.
func (c *ConfYaml) SanitizedCopy() *ConfYaml {
	sanitized := *c

	// Core TLS & proxy
	sanitized.Core.CertBase64 = redact(c.Core.CertBase64)
	sanitized.Core.KeyBase64 = redact(c.Core.KeyBase64)
	sanitized.Core.HTTPProxy = redact(c.Core.HTTPProxy)
	sanitized.Core.CertPath = redact(c.Core.CertPath)
	sanitized.Core.KeyPath = redact(c.Core.KeyPath)
	if len(c.Core.FeedbackHeader) > 0 {
		sanitized.Core.FeedbackHeader = make([]string, len(c.Core.FeedbackHeader))
		for i, v := range c.Core.FeedbackHeader {
			if v != "" {
				sanitized.Core.FeedbackHeader[i] = "[REDACTED]"
			}
		}
	}

	// Android
	sanitized.Android.KeyPath = redact(c.Android.KeyPath)
	sanitized.Android.Credential = redact(c.Android.Credential)

	// Huawei
	sanitized.Huawei.AppSecret = redact(c.Huawei.AppSecret)

	// iOS
	sanitized.Ios.KeyPath = redact(c.Ios.KeyPath)
	sanitized.Ios.KeyBase64 = redact(c.Ios.KeyBase64)
	sanitized.Ios.Password = redact(c.Ios.Password)
	sanitized.Ios.KeyID = redact(c.Ios.KeyID)
	sanitized.Ios.TeamID = redact(c.Ios.TeamID)

	// Queue Redis
	sanitized.Queue.Redis.Username = redact(c.Queue.Redis.Username)
	sanitized.Queue.Redis.Password = redact(c.Queue.Redis.Password)

	// Stat Redis
	sanitized.Stat.Redis.Username = redact(c.Stat.Redis.Username)
	sanitized.Stat.Redis.Password = redact(c.Stat.Redis.Password)

	return &sanitized
}

func redact(s string) string {
	if s != "" {
		return "[REDACTED]"
	}
	return ""
}
