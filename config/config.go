package config

import (
	"flag"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

const (
	configFileName = "camino-messenger-provider.yaml"

	configFlagKey = "config"
)

type SupportedRequestTypesFlag []string

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
	Port           int    `mapstructure:"rpc-server-port"`
	Unencrypted    bool   `mapstructure:"rpc-unencrypted"`
	ServerCertFile string `mapstructure:"server-cert-file"`
	ServerKeyFile  string `mapstructure:"server-key-file"`
}
type PartnerPluginConfig struct {
	PartnerPluginHost string `mapstructure:"partner-plugin-host"`
	PartnerPluginPort int    `mapstructure:"partner-plugin-port"`
}
type ProcessorConfig struct {
	Timeout               int                       `mapstructure:"response-timeout"` // in milliseconds
	SupportedRequestTypes SupportedRequestTypesFlag `mapstructure:"supported-request-types"`
}
type Config struct {
	AppConfig           `mapstructure:",squash"`
	MatrixConfig        `mapstructure:",squash"`
	RPCServerConfig     `mapstructure:",squash"`
	PartnerPluginConfig `mapstructure:",squash"`
	ProcessorConfig     `mapstructure:",squash"`
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
	readMessengerConfig(cfg.ProcessorConfig, fs)

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

func (i *SupportedRequestTypesFlag) String() string {
	return "[" + strings.Join(*i, ",") + "]"
}
func (i *SupportedRequestTypesFlag) Contains(requestType string) bool {
	return slices.Contains(*i, requestType)
}

func (i *SupportedRequestTypesFlag) Set(requestType string) error {
	*i = append(*i, requestType)
	return nil
}
