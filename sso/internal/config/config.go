package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

type Config struct {
	Env            string     `json:"env" yaml:"env" env-default:"local"`
	DSN            string     `env:"DSN_STRING"`
	GRPC           GRPCConfig `yaml:"grpc"`
	MigrationsPath string
	TokenTTL       time.Duration `yaml:"token_ttl" env-default:"1h"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

// Be careful with panics, as we used them only in app launching
func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty!")
	}

	// check for file existence
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("config path is empty: " + err.Error())
	}

	return &cfg
}

// Fetches config path from command line flag or enviroment variable
// Priority: flag > env > default (empty string)
func fetchConfigPath() string {
	var res string

	// "--config" is flag name
	flag.StringVar(&res, "config", "", "path to config file")
	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}
	flag.Parse()

	return res
}
