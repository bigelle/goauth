package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Port string `env:"PORT" env-default:"8080"`
	Host string `env:"HOST" env-default:"localhost"`
}

func SetupConfig() (*Config, error) {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	return &cfg, err
}
