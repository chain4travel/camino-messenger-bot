package config

import (
	"crypto/ecdsa"
	"flag"
	"net/url"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	envVarPrefix  = "CMB" // CMB=camino-messenger-bot
	configFlagKey = "config"
)

type UnparsedConfig struct {
}

func (uc *UnparsedConfig) Parse() (*Config, error) {
	return &Config{}, nil
}

type Config struct {
	DeveloperMode bool `mapstructure:"developer_mode"`

	BotKey                              *ecdsa.PrivateKey `mapstructure:"evm_private_key"`
	CChainRPCURL                        url.URL           `mapstructure:"rpc_url"`
	BookingTokenAddress                 common.Address    `mapstructure:"booking_token_address"`
	CMAccountAddress                    common.Address    `mapstructure:"cm_account_address"`
	NetworkFeeRecipientBotAddress       common.Address    `mapstructure:"network_fee_recipient_bot_address"`
	NetworkFeeRecipientCMAccountAddress common.Address    `mapstructure:"network_fee_recipient_cm_account"`
	ChequeExpirationTime                uint64            `mapstructure:"cheque_expiration_time"`
	MinChequeDurationUntilExpiration    uint64            `mapstructure:"min_cheque_duration_until_expiration"` // seconds
	CashInPeriod                        uint64            `mapstructure:"cash_in_period"`                       // seconds

	MatrixHost   url.URL `mapstructure:"matrix_host"`
	MatrixDBPath string  `mapstructure:"matrix_store"`

	RPCPort           int    `mapstructure:"rpc_server_port"`
	RPCUnencrypted    bool   `mapstructure:"rpc_unencrypted"`
	RPCServerCertFile string `mapstructure:"rpc_server_cert_file"`
	RPCServerKeyFile  string `mapstructure:"rpc_server_key_file"`

	PartnerPluginHost url.URL `mapstructure:"partner_plugin_host"`
	Unencrypted       bool    `mapstructure:"partner_plugin_unencrypted"`
	CACertFile        string  `mapstructure:"partner_plugin_ca_file"`

	ResponseTimeout int `mapstructure:"response_timeout"` // in milliseconds

	TracingEnabled  bool    `mapstructure:"tracing_enabled"`
	TracingHost     url.URL `mapstructure:"tracing_host"`
	TracingInsecure bool    `mapstructure:"tracing_insecure"`
	TracingCertFile string  `mapstructure:"tracing_cert_file"`
	TracingKeyFile  string  `mapstructure:"tracing_key_file"`

	DBPath         string `mapstructure:"db_path"`
	DBName         string `mapstructure:"db_name"`
	MigrationsPath string `mapstructure:"migrations_path"`
}

func ReadConfig() (*UnparsedConfig, error) {
	var configFile string

	// Define command-line flags
	flag.StringVar(&configFile, configFlagKey, "", "Path to configuration file")
	flag.Parse()

	viper.New()
	viper.SetConfigFile(configFile)

	// Enable reading from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix(envVarPrefix)

	cfg := &UnparsedConfig{}
	fs := flag.NewFlagSet("tcm", flag.ExitOnError)
	readAppConfig(cfg.AppConfig, fs)
	readMatrixConfig(cfg.MatrixConfig, fs)
	readRPCServerConfig(cfg.RPCServerConfig, fs)
	readPartnerRPCServerConfig(cfg.PartnerPluginConfig, fs)
	readMessengerConfig(cfg.ProcessorConfig, fs)
	readEvmConfig(cfg.EvmConfig, fs)
	readTracingConfig(cfg.TracingConfig, fs)
	readDBConfig(cfg.DBConfig, fs)

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
