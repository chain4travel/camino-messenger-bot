package config

import (
	"flag"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

const (
	envVarPrefix  = "CMB" // CMB=camino-messenger-bot
	configFlagKey = "config"
)

type SupportedRequestTypesFlag []string

type AppConfig struct {
	DeveloperMode         bool                      `mapstructure:"developer_mode"`
	SupportedRequestTypes SupportedRequestTypesFlag `mapstructure:"supported_request_types"`
}
type MatrixConfig struct {
	Username string `mapstructure:"matrix_username"`
	Password string `mapstructure:"matrix_password"`
	Host     string `mapstructure:"matrix_host"`
	Store    string `mapstructure:"matrix_store"`
}
type RPCServerConfig struct {
	Port           int    `mapstructure:"rpc_server_port"`
	Unencrypted    bool   `mapstructure:"rpc_unencrypted"`
	ServerCertFile string `mapstructure:"rpc_server_cert_file"`
	ServerKeyFile  string `mapstructure:"rpc_server_key_file"`
}
type PartnerPluginConfig struct {
	Host        string `mapstructure:"partner_plugin_host"`
	Port        int    `mapstructure:"partner_plugin_port"`
	Unencrypted bool   `mapstructure:"partner_plugin_unencrypted"`
	CACertFile  string `mapstructure:"partner_plugin_ca_file"`
}
type ProcessorConfig struct {
	Timeout int `mapstructure:"messenger_timeout"` // in milliseconds
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
	flag.StringVar(&configFile, configFlagKey, "", "Path to configuration file")
	flag.Parse()

	viper.New()
	viper.SetConfigFile(configFile)

	// Enable reading from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix(envVarPrefix)

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

	viper.ReadInConfig() // ignore config-file-reading-errors as we have env vars as fallback configuration

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
