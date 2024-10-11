package config

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ******* Parsed config *******
//
//

type Config struct {
	DeveloperMode bool

	BotKey           *ecdsa.PrivateKey
	CMAccountAddress common.Address

	ChainRPCURL         url.URL
	BookingTokenAddress common.Address

	NetworkFeeRecipientBotAddress       common.Address
	NetworkFeeRecipientCMAccountAddress common.Address

	ChequeExpirationTime             *big.Int // seconds
	MinChequeDurationUntilExpiration *big.Int // seconds
	CashInPeriod                     time.Duration

	ResponseTimeout time.Duration

	RPCServer     RPCServerConfig
	PartnerPlugin PartnerPluginConfig
	Tracing       TracingConfig
	DB            SQLiteDBConfig
	Matrix        MatrixConfig
}

type TracingConfig struct {
	Enabled  bool
	HostURL  url.URL
	Insecure bool
	CertFile string
	KeyFile  string
}

type PartnerPluginConfig struct {
	Enabled     bool
	HostURL     url.URL
	Unencrypted bool
	CACertFile  string
}

type MatrixConfig struct {
	HostURL url.URL
	Store   string
}

type SQLiteDBConfig struct {
	Common        UnparsedSQLiteDBConfig
	Scheduler     UnparsedSQLiteDBConfig
	ChequeHandler UnparsedSQLiteDBConfig
}

// ******* Common *******
//
//

type RPCServerConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Port           uint64 `mapstructure:"port"`
	Unencrypted    bool   `mapstructure:"unencrypted"`
	ServerCertFile string `mapstructure:"cert_file"`
	ServerKeyFile  string `mapstructure:"key_file"`
}

// ******* Unparsed config *******
//
//

type UnparsedConfig struct {
	DeveloperMode bool `mapstructure:"developer_mode"`

	BotKey           string `mapstructure:"bot_key"`
	CMAccountAddress string `mapstructure:"cm_account_address"`

	ChainRPCURL         string `mapstructure:"chain_rpc_url"`
	BookingTokenAddress string `mapstructure:"booking_token_address"`

	NetworkFeeRecipientBotAddress       string `mapstructure:"network_fee_recipient_bot_address"`
	NetworkFeeRecipientCMAccountAddress string `mapstructure:"network_fee_recipient_cm_account"`

	ChequeExpirationTime             uint64 `mapstructure:"cheque_expiration_time"`               // seconds
	MinChequeDurationUntilExpiration uint64 `mapstructure:"min_cheque_duration_until_expiration"` // seconds
	CashInPeriod                     int64  `mapstructure:"cash_in_period"`                       // seconds

	ResponseTimeout int64 `mapstructure:"response_timeout"` // milliseconds

	PartnerPlugin UnparsedPartnerPluginConfig `mapstructure:"partner_plugin"`
	Tracing       UnparsedTracingConfig       `mapstructure:"tracing"`
	Matrix        UnparsedMatrixConfig        `mapstructure:"matrix"`

	RPCServer RPCServerConfig        `mapstructure:"rpc_server"`
	DB        UnparsedSQLiteDBConfig `mapstructure:"db"`
}

type UnparsedTracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Insecure bool   `mapstructure:"insecure"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type UnparsedPartnerPluginConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Host        string `mapstructure:"host"`
	Unencrypted bool   `mapstructure:"unencrypted"`
	CACertFile  string `mapstructure:"ca_file"`
}

type UnparsedMatrixConfig struct {
	Host  string `mapstructure:"host"`
	Store string `mapstructure:"store"`
}

type UnparsedSQLiteDBConfig struct {
	DBPath         string `mapstructure:"path"`
	MigrationsPath string `mapstructure:"migrations_path"`
}

func (cfg *Config) unparse() *UnparsedConfig {
	return &UnparsedConfig{
		DB:        cfg.DB.Common,
		RPCServer: cfg.RPCServer,
		Tracing: UnparsedTracingConfig{
			Enabled:  cfg.Tracing.Enabled,
			Host:     cfg.Tracing.HostURL.String(),
			Insecure: cfg.Tracing.Insecure,
			CertFile: cfg.Tracing.CertFile,
			KeyFile:  cfg.Tracing.KeyFile,
		},
		PartnerPlugin: UnparsedPartnerPluginConfig{
			Enabled:     cfg.PartnerPlugin.Enabled,
			Host:        cfg.PartnerPlugin.HostURL.String(),
			Unencrypted: cfg.PartnerPlugin.Unencrypted,
			CACertFile:  cfg.PartnerPlugin.CACertFile,
		},
		Matrix: UnparsedMatrixConfig{
			Host:  cfg.Matrix.HostURL.String(),
			Store: cfg.Matrix.Store,
		},
		DeveloperMode:                       cfg.DeveloperMode,
		BotKey:                              hex.EncodeToString(crypto.FromECDSA(cfg.BotKey)),
		CMAccountAddress:                    cfg.CMAccountAddress.Hex(),
		ChainRPCURL:                         cfg.ChainRPCURL.String(),
		BookingTokenAddress:                 cfg.BookingTokenAddress.Hex(),
		NetworkFeeRecipientBotAddress:       cfg.NetworkFeeRecipientBotAddress.Hex(),
		NetworkFeeRecipientCMAccountAddress: cfg.NetworkFeeRecipientCMAccountAddress.Hex(),
		ChequeExpirationTime:                cfg.ChequeExpirationTime.Uint64(),
		MinChequeDurationUntilExpiration:    cfg.MinChequeDurationUntilExpiration.Uint64(),
		CashInPeriod:                        int64(cfg.CashInPeriod / time.Second),
		ResponseTimeout:                     int64(cfg.ResponseTimeout / time.Millisecond),
	}
}
