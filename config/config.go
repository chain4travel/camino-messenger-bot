package config

import (
	"crypto/ecdsa"
	"math/big"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

//
// ******* Parsed config *******
//

type Config struct {
	DeveloperMode bool

	BotKey           *ecdsa.PrivateKey
	CMAccountAddress common.Address

	CChainRPCURL        url.URL
	BookingTokenAddress common.Address

	NetworkFeeRecipientBotAddress       common.Address
	NetworkFeeRecipientCMAccountAddress common.Address

	ChequeExpirationTime             *big.Int // seconds
	MinChequeDurationUntilExpiration *big.Int // seconds
	CashInPeriod                     time.Duration

	ResponseTimeout time.Duration

	RPCServerConfig
	PartnerPluginConfig
	TracingConfig
	DBConfig
	MatrixConfig
}

type TracingConfig struct {
	Enabled  bool
	HostURL  *url.URL
	Insecure bool
	CertFile string
	KeyFile  string
}

type DBConfig struct {
	DBPath         string
	DBName         string
	MigrationsPath string
}
type PartnerPluginConfig struct {
	HostURL     *url.URL
	Unencrypted bool
	CACertFile  string
}

type MatrixConfig struct {
	HostURL url.URL
	Store   string
}

type RPCServerConfig struct {
	Port           uint64
	Unencrypted    bool
	ServerCertFile string
	ServerKeyFile  string
}

//
// ******* Unparsed config *******
//

type UnparsedConfig struct {
	DeveloperMode bool `mapstructure:"developer_mode"`

	BotKey           string `mapstructure:"evm_private_key"`
	CMAccountAddress string `mapstructure:"cm_account_address"`

	CChainRPCURL        string `mapstructure:"rpc_url"`
	BookingTokenAddress string `mapstructure:"booking_token_address"`

	NetworkFeeRecipientBotAddress       string `mapstructure:"network_fee_recipient_bot_address"`
	NetworkFeeRecipientCMAccountAddress string `mapstructure:"network_fee_recipient_cm_account"`

	ChequeExpirationTime             uint64 `mapstructure:"cheque_expiration_time"`               // seconds
	MinChequeDurationUntilExpiration uint64 `mapstructure:"min_cheque_duration_until_expiration"` // seconds
	CashInPeriod                     int64  `mapstructure:"cash_in_period"`                       // seconds

	ResponseTimeout int64 `mapstructure:"response_timeout"` // milliseconds

	UnparsedPartnerPluginConfig
	UnparsedTracingConfig

	RPCServerConfig
	MatrixConfig
	DBConfig
}

type UnparsedTracingConfig struct {
	Enabled  bool   `mapstructure:"tracing_enabled"`
	Host     string `mapstructure:"tracing_host"`
	Insecure bool   `mapstructure:"tracing_insecure"`
	CertFile string `mapstructure:"tracing_cert_file"`
	KeyFile  string `mapstructure:"tracing_key_file"`
}

type UnparsedPartnerPluginConfig struct {
	Host        string `mapstructure:"partner_plugin_host"`
	Unencrypted bool   `mapstructure:"partner_plugin_unencrypted"`
	CACertFile  string `mapstructure:"partner_plugin_ca_file"`
}

type UnparsedMatrixConfig struct {
	Host  string `mapstructure:"matrix_host"`
	Store string `mapstructure:"matrix_store"`
}
