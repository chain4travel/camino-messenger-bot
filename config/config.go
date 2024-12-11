// Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
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

	ChainRPCURL         string
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

type SQLiteDBConfig struct {
	Common        UnparsedSQLiteDBConfig
	Scheduler     UnparsedSQLiteDBConfig
	ChequeHandler UnparsedSQLiteDBConfig
}

// ******* Common *******
//
//

type TracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Insecure bool   `mapstructure:"insecure"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type PartnerPluginConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Host        string `mapstructure:"host"`
	Unencrypted bool   `mapstructure:"unencrypted"`
	CACertFile  string `mapstructure:"ca_file"`
}

type MatrixConfig struct {
	Host  string `mapstructure:"host"`
	Store string
}

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

	PartnerPlugin PartnerPluginConfig `mapstructure:"partner_plugin"`
	Tracing       TracingConfig       `mapstructure:"tracing"`
	RPCServer     RPCServerConfig     `mapstructure:"rpc_server"`

	Matrix UnparsedMatrixConfig   `mapstructure:"matrix"`
	DB     UnparsedSQLiteDBConfig `mapstructure:"db"`
}

type UnparsedSQLiteDBConfig struct {
	DBPath         string `mapstructure:"path"`
	MigrationsPath string `mapstructure:"migrations_path"`
}

type UnparsedMatrixConfig struct {
	Host string `mapstructure:"host"`
}

func (cfg *Config) unparse() *UnparsedConfig {
	return &UnparsedConfig{
		DB:            cfg.DB.Common,
		RPCServer:     cfg.RPCServer,
		Tracing:       cfg.Tracing,
		PartnerPlugin: cfg.PartnerPlugin,
		Matrix: UnparsedMatrixConfig{
			Host: cfg.Matrix.Host,
		},
		DeveloperMode:                       cfg.DeveloperMode,
		BotKey:                              hex.EncodeToString(crypto.FromECDSA(cfg.BotKey)),
		CMAccountAddress:                    cfg.CMAccountAddress.Hex(),
		ChainRPCURL:                         cfg.ChainRPCURL,
		BookingTokenAddress:                 cfg.BookingTokenAddress.Hex(),
		NetworkFeeRecipientBotAddress:       cfg.NetworkFeeRecipientBotAddress.Hex(),
		NetworkFeeRecipientCMAccountAddress: cfg.NetworkFeeRecipientCMAccountAddress.Hex(),
		ChequeExpirationTime:                cfg.ChequeExpirationTime.Uint64(),
		MinChequeDurationUntilExpiration:    cfg.MinChequeDurationUntilExpiration.Uint64(),
		CashInPeriod:                        int64(cfg.CashInPeriod / time.Second),
		ResponseTimeout:                     int64(cfg.ResponseTimeout / time.Millisecond),
	}
}
