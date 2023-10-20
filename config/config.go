package config

import (
	"flag"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
)

const (
	AppName = "camino-messenger-provider"
)

type MatrixConfig struct {
	Username   string
	Password   string
	MatrixHost string
}
type RPCServerConfig struct {
	RPCServerPort int
}
type PartnerPluginConfig struct {
	PartnerPluginHost string
	PartnerPluginPort int
}
type Config struct {
	MatrixConfig
	RPCServerConfig
	PartnerPluginConfig
}

func ReadConfig() (*Config, error) {
	fs := flag.NewFlagSet("tcm", flag.ExitOnError)
	cfg := &Config{}

	err := fs.Parse(os.Args[:])
	if err != nil {
		return nil, err
	}

	// Use viper to bind command-line flags to config
	v := viper.New()
	readMatrixConfig(cfg.MatrixConfig, fs)
	readRPCServerConfig(cfg.RPCServerConfig, fs)
	readPartnerRpcServerConfig(cfg.PartnerPluginConfig, fs)

	pfs := pflag.NewFlagSet(fs.Name(), pflag.ContinueOnError)
	pfs.AddGoFlagSet(fs)
	err = v.BindPFlags(pfs)
	if err != nil {
		return nil, err
	}

	// Check for missing required arguments
	if cfg.Username == "" || cfg.Password == "" || cfg.MatrixHost == "" {
		return nil, fmt.Errorf("missing required arguments")
	}

	return cfg, nil
}
