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
	Key   string `mapstructure:"matrix_key"` // TODO @evlekht I'd suggest to add some parsed config, so we'll see on config read if some fields are invalid
	Host  string `mapstructure:"matrix_host"`
	Store string `mapstructure:"matrix_store"`
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
	Timeout int `mapstructure:"response_timeout"` // in milliseconds
}

type EvmConfig struct {
	PrivateKey          string `mapstructure:"evm_private_key"`
	RPCURL              string `mapstructure:"rpc_url"`
	SupplierName        string `mapstructure:"supplier_name"`
	BookingTokenAddress string `mapstructure:"booking_token_address"`
	BuyableUntilDefault uint64 `mapstructure:"buyable_until_default"`
	CMAccountAddress    string `mapstructure:"cm_account_address"`
	CMNetworkFee        uint64 `mapstructure:"cm_network_fee"`
	DomainName          string `mapstructure:"domain_name"`
	DomainVersion       uint64 `mapstructure:"domain_version"`
}

type TracingConfig struct {
	Enabled  bool   `mapstructure:"tracing_enabled"`
	Host     string `mapstructure:"tracing_host"`
	Port     int    `mapstructure:"tracing_port"`
	Insecure bool   `mapstructure:"tracing_insecure"`
	CertFile string `mapstructure:"tracing_cert_file"`
	KeyFile  string `mapstructure:"tracing_key_file"`
}
type Config struct {
	AppConfig           `mapstructure:",squash"` // TODO use nested yaml structure
	EvmConfig           `mapstructure:",squash"`
	MatrixConfig        `mapstructure:",squash"`
	RPCServerConfig     `mapstructure:",squash"`
	PartnerPluginConfig `mapstructure:",squash"`
	ProcessorConfig     `mapstructure:",squash"`
	TracingConfig       `mapstructure:",squash"`
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
	readPartnerRPCServerConfig(cfg.PartnerPluginConfig, fs)
	readMessengerConfig(cfg.ProcessorConfig, fs)
	readEvmConfig(cfg.EvmConfig, fs)
	readTracingConfig(cfg.TracingConfig, fs)

	// Parse command-line flags
	pfs := pflag.NewFlagSet(fs.Name(), pflag.ContinueOnError)
	pfs.AddGoFlagSet(fs)
	err := viper.BindPFlags(pfs)
	if err != nil {
		return nil, err
	}

	// read configuration file if provided, otherwise rely on env vars
	if configFile != "" {
		if err := viper.ReadInConfig(); err != nil {
			return cfg, err
		}
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
