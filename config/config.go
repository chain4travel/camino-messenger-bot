package config

import (
	"flag"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	configFileName = "camino-messenger-provider.yaml"

	configFlagKey = "config"
)

type MatrixConfig struct {
	Username   string `mapstructure:"matrix-username"`
	Password   string `mapstructure:"matrix-password"`
	MatrixHost string `mapstructure:"matrix-host"`
}
type RPCServerConfig struct {
	RPCServerPort int `mapstructure:"rpc-server-port"`
}
type PartnerPluginConfig struct {
	PartnerPluginHost string `mapstructure:"partner-plugin-host"`
	PartnerPluginPort int    `mapstructure:"partner-plugin-port"`
}
type Config struct {
	MatrixConfig        `mapstructure:",squash"`
	RPCServerConfig     `mapstructure:",squash"`
	PartnerPluginConfig `mapstructure:",squash"`
}

func ReadConfig() (*Config, error) {
	var configFile string

	// Define command-line flags
	flag.StringVar(&configFile, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	viper.New()
	viper.SetConfigFile(configFile)

	cfg := &Config{}
	fs := flag.NewFlagSet("tcm", flag.ExitOnError)
	readMatrixConfig(cfg.MatrixConfig, fs)
	readRPCServerConfig(cfg.RPCServerConfig, fs)
	readPartnerRpcServerConfig(cfg.PartnerPluginConfig, fs)
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

	fmt.Println(cfg)
	return cfg, nil
}
