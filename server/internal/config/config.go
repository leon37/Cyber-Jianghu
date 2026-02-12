package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	AI       AIConfig       `yaml:"ai"`
	Memory   MemoryConfig   `yaml:"memory"`
	Live     LiveConfig     `yaml:"live"`
	Queue    QueueConfig    `yaml:"queue"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	MySQL  MySQLConfig  `yaml:"mysql"`
	Redis  RedisConfig  `yaml:"redis"`
	Qdrant QdrantConfig `yaml:"qdrant"`
}

type MySQLConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type QdrantConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	APIKey     string `yaml:"api_key"`
	Collection string `yaml:"collection"`
	VectorSize int    `yaml:"vector_size"`
}

type AIConfig struct {
	GLM5      GLM5Config      `yaml:"glm5"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	ComfyUI   ComfyUIConfig   `yaml:"comfyui"`
	SoVITS    SoVITSConfig    `yaml:"sovits"`
}

type GLM5Config struct {
	BaseURL     string  `yaml:"base_url"`
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

type EmbeddingConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	APIKey   string `yaml:"api_key"`
}

type ComfyUIConfig struct {
	BaseURL      string        `yaml:"base_url"`
	WorkflowFile string        `yaml:"workflow_file"`
	Timeout      time.Duration `yaml:"timeout"`
}

type SoVITSConfig struct {
	BaseURL   string        `yaml:"base_url"`
	ModelPath string        `yaml:"model_path"`
	Timeout   time.Duration `yaml:"timeout"`
}

type MemoryConfig struct {
	RetentionDays         int `yaml:"retention_days"`
	MaxMemoriesPerSession int `yaml:"max_memories_per_session"`
	SearchLimit           int `yaml:"search_limit"`
}

type LiveConfig struct {
	Bilibili BilibiliConfig `yaml:"bilibili"`
	Douyin   DouyinConfig   `yaml:"douyin"`
}

type BilibiliConfig struct {
	RoomID            string        `yaml:"room_id"`
	Cookie            string        `yaml:"cookie"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

type DouyinConfig struct {
	RoomID            string        `yaml:"room_id"`
	Cookie            string        `yaml:"cookie"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

type QueueConfig struct {
	MaxWorkers   int `yaml:"max_workers"`
	MaxQueueSize int `yaml:"max_queue_size"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply environment variable overrides
	if apiKey := os.Getenv("ZHIPUAI_API_KEY"); apiKey != "" {
		cfg.AI.GLM5.APIKey = apiKey
		cfg.AI.Embedding.APIKey = apiKey
	}
	if apiKey := os.Getenv("QDRANT_API_KEY"); apiKey != "" {
		cfg.Database.Qdrant.APIKey = apiKey
	}

	return &cfg, nil
}
