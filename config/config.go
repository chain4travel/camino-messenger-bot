package config

import (
	"flag"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	envVarPrefix  = "CMB" // CMB=camino-messenger-bot
	configFlagKey = "config"
)

type AppConfig struct {
	DeveloperMode bool `mapstructure:"developer_mode"`
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

// should MessengerCashier related config be here?
type EvmConfig struct {
	PrivateKey                          string `mapstructure:"evm_private_key"`
	RPCURL                              string `mapstructure:"rpc_url"`
	SupplierName                        string `mapstructure:"supplier_name"`
	BookingTokenAddress                 string `mapstructure:"booking_token_address"`
	CMAccountAddress                    string `mapstructure:"cm_account_address"`
	NetworkFeeRecipientBotAddress       string `mapstructure:"network_fee_recipient_bot_address"`
	NetworkFeeRecipientCMAccountAddress string `mapstructure:"network_fee_recipient_cm_account"`
	ChequeExpirationTime                uint64 `mapstructure:"cheque_expiration_time"`
	MinChequeDurationUntilExpiration    uint64 `mapstructure:"min_cheque_duration_until_expiration"` // seconds
	CashInPeriod                        uint64 `mapstructure:"cash_in_period"`                       // seconds
}

type DBConfig struct {
	DBPath         string `mapstructure:"db_path"`
	DBName         string `mapstructure:"db_name"`
	MigrationsPath string `mapstructure:"migrations_path"`
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
	DBConfig            `mapstructure:",squash"`
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
