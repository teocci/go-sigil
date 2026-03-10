package config

import (
	"fmt"
	"time"

	"go-sigil/internal/constants"

	"github.com/spf13/viper"
)

// Config holds all Sigil configuration.
type Config struct {
	CacheRoot string
	LogLevel  string
	LogFile   string

	Security   SecurityConfig
	Indexing   IndexingConfig
	Enrichment EnrichmentConfig
}

// SecurityConfig controls file filtering behavior.
type SecurityConfig struct {
	MaxFileSizeBytes     int64    `mapstructure:"max_file_size_bytes"`
	BinaryNullThreshold  float64  `mapstructure:"binary_null_threshold"`
	ExtraSecretFilenames []string `mapstructure:"extra_secret_filenames"`
	ExtraIgnoreFilenames []string `mapstructure:"extra_ignore_filenames"`
	ExtraSecretPatterns  []string `mapstructure:"extra_secret_patterns"`
}

// IndexingConfig controls indexing limits.
type IndexingConfig struct {
	MaxFiles int `mapstructure:"max_files"`
}

// EnrichmentConfig controls LLM enrichment behavior.
type EnrichmentConfig struct {
	BatchSize int           `mapstructure:"batch_size"`
	Timeout   time.Duration `mapstructure:"timeout"`
	Disabled  bool          `mapstructure:"disabled"`
}

// Load reads configuration from ~/.sigil/config.toml, environment variables,
// and applies defaults. Environment variables take precedence over the config file.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("cache_root", DefaultCacheRoot())
	v.SetDefault("log_level", "WARNING")
	v.SetDefault("log_file", "")

	v.SetDefault("security.max_file_size_bytes", constants.DefaultMaxFileSize)
	v.SetDefault("security.binary_null_threshold", constants.DefaultBinaryNullThresh)
	v.SetDefault("security.extra_secret_filenames", []string{})
	v.SetDefault("security.extra_ignore_filenames", []string{})
	v.SetDefault("security.extra_secret_patterns", []string{})

	v.SetDefault("indexing.max_files", constants.DefaultMaxIndexFiles)

	v.SetDefault("enrichment.batch_size", constants.DefaultEnrichBatchSize)
	v.SetDefault("enrichment.timeout", constants.DefaultEnrichTimeout)
	v.SetDefault("enrichment.disabled", false)

	// Environment variable mapping
	v.SetEnvPrefix("")
	v.BindEnv("cache_root", "CODE_INDEX_PATH")
	v.BindEnv("log_level", "SIGIL_LOG_LEVEL")
	v.BindEnv("log_file", "SIGIL_LOG_FILE")
	v.BindEnv("indexing.max_files", "SIGIL_MAX_INDEX_FILES")
	v.BindEnv("enrichment.batch_size", "SIGIL_ENRICH_BATCH_SIZE")

	// Config file
	cacheRoot := v.GetString("cache_root")
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(cacheRoot)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// Config file not found is fine — use defaults + env
	}

	cfg := &Config{
		CacheRoot: v.GetString("cache_root"),
		LogLevel:  v.GetString("log_level"),
		LogFile:   v.GetString("log_file"),
	}

	if err := v.UnmarshalKey("security", &cfg.Security); err != nil {
		return nil, fmt.Errorf("parse security config: %w", err)
	}
	if err := v.UnmarshalKey("indexing", &cfg.Indexing); err != nil {
		return nil, fmt.Errorf("parse indexing config: %w", err)
	}
	if err := v.UnmarshalKey("enrichment", &cfg.Enrichment); err != nil {
		return nil, fmt.Errorf("parse enrichment config: %w", err)
	}

	return cfg, nil
}
