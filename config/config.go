package config

import (
	"flag"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	configFileName = "camino-messenger-provider.yaml"

	configFlagKey = "config"
)

type AppConfig struct {
	DeveloperMode bool `mapstructure:"developer-mode"`
}
type MatrixConfig struct {
	Username string `mapstructure:"matrix-username"`
	Password string `mapstructure:"matrix-password"`
	Host     string `mapstructure:"matrix-host"`
	Store    string `mapstructure:"matrix-store"`
}
type RPCServerConfig struct {
	RPCServerPort int `mapstructure:"rpc-server-port"`
}
type PartnerPluginConfig struct {
	PartnerPluginHost string `mapstructure:"partner-plugin-host"`
	PartnerPluginPort int    `mapstructure:"partner-plugin-port"`
}
type MessengerConfig struct {
	Timeout int `mapstructure:"response-timeout"` // in milliseconds
}
type Config struct {
	AppConfig           `mapstructure:",squash"`
	MatrixConfig        `mapstructure:",squash"`
	RPCServerConfig     `mapstructure:",squash"`
	PartnerPluginConfig `mapstructure:",squash"`
	MessengerConfig     `mapstructure:",squash"`
}

func ReadConfig() (*Config, error) {
	var configFile string

	// Define command-line flags
	flag.StringVar(&configFile, configFlagKey, configFileName, "Path to configuration file")
	flag.Parse()

	viper.New()
	viper.SetConfigFile(configFile)

	cfg := &Config{}
	fs := flag.NewFlagSet("tcm", flag.ExitOnError)
	readAppConfig(cfg.AppConfig, fs)
	readMatrixConfig(cfg.MatrixConfig, fs)
	readRPCServerConfig(cfg.RPCServerConfig, fs)
	readPartnerRpcServerConfig(cfg.PartnerPluginConfig, fs)
	readMessengerConfig(cfg.MessengerConfig, fs)

	// Parse command-line flags
	pfs := pflag.NewFlagSet(fs.Name(), pflag.ContinueOnError)
	pfs.AddGoFlagSet(fs)
	err := viper.BindPFlags(pfs)
	if err != nil {
		return nil, err
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
