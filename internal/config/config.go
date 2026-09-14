package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	configSchemaResourceName = "config.schema.yaml"
)

//go:embed config.schema.yaml
var embeddedJsonSchema []byte

var (
	schemaDoc interface{}
	schema    *jsonschema.Schema
)

func init() {
	var err error

	if err = yaml.Unmarshal(embeddedJsonSchema, &schemaDoc); err != nil {
		panic(err)
	}
	c := jsonschema.NewCompiler()
	if err = c.AddResource("config.schema.yaml", schemaDoc); err != nil {
		panic(err)
	}

	schema, err = c.Compile("config.schema.yaml")
	if err != nil {
		panic(err)
	}

}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Driver string       `mapstructure:"driver"`
	Sqlite SqliteConfig `mapstructure:"sqlite"`
}

type SqliteConfig struct {
	File  string `mapstructure:"file"`
	Cache string `mapstructure:"cache"`
}

func LoadConfig(file ...string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	setupDefaults(v)

	cfgFile := "config.yaml"
	if file != nil {
		cfgFile = file[0]
	}
	v.SetConfigFile(cfgFile)

	if err := v.ReadInConfig(); err != nil {
		if !isMissingConfigFile(err) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		return DefaultConfig(), nil
	}

	rawMap := v.AllSettings()

	if err := schema.Validate(rawMap); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error binding config: %w", err)
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	v := viper.New()
	setupDefaults(v)

	var cfg Config
	// can it throw an error?
	if err := v.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "viper unmarshal error: %s", err.Error())
	}
	return &cfg
}

func setupDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("server.port", 50051)
}

func isMissingConfigFile(err error) bool {
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		return true
	}
	if os.IsNotExist(err) {
		return true
	}
	return false
}
