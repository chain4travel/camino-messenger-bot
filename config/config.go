// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
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
	E2ETestMode   bool

	BotKey           *ecdsa.PrivateKey
	CMAccountAddress common.Address

	ChainRPCURL         string
	BookingTokenAddress common.Address

	NetworkFeeRecipientBotAddress       common.Address
	NetworkFeeRecipientCMAccountAddress common.Address

	ChequeExpirationTime             *big.Int // seconds
	MinChequeDurationUntilExpiration *big.Int // seconds
	MaxAllowedServiceFee             *big.Int // aCAM

	ResponseTimeout time.Duration

	RecordExpiration bool

	RPCServer     RPCServerConfig
	PartnerPlugin PartnerPluginConfig
	DB            SQLiteDBConfig
	Matrix        MatrixConfig
	CashIn        CashInConfig // TODO@ keep naming consistent with other cash-in config structs and flags
}

type SQLiteDBConfig struct {
	Common                 UnparsedSQLiteDBConfig
	Scheduler              UnparsedSQLiteDBConfig
	ChequeHandler          UnparsedSQLiteDBConfig
	EventListener          UnparsedSQLiteDBConfig
	MessagesEncoderDecoder UnparsedSQLiteDBConfig
}

type CashInConfig struct { // TODO@ keep naming consistent with other cash-in config structs and flags
	Period    time.Duration `mapstructure:"period"`
	MinAmount uint64        `mapstructure:"min_amount"` // aCAM // TODO@ clarify if its aCAM; maybe be do big int
}

// ******* Common *******
//
//

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
	E2ETestMode   bool `mapstructure:"e2e_test_mode"`

	BotKey           string `mapstructure:"bot_key"`
	CMAccountAddress string `mapstructure:"cm_account_address"`

	ChainRPCURL         string `mapstructure:"chain_rpc_url"`
	BookingTokenAddress string `mapstructure:"booking_token_address"`

	NetworkFeeRecipientBotAddress       string `mapstructure:"network_fee_recipient_bot_address"`
	NetworkFeeRecipientCMAccountAddress string `mapstructure:"network_fee_recipient_cm_account"`

	ChequeExpirationTime             uint64 `mapstructure:"cheque_expiration_time"`               // seconds
	MinChequeDurationUntilExpiration uint64 `mapstructure:"min_cheque_duration_until_expiration"` // seconds
	MaxAllowedServiceFee             string `mapstructure:"max_allowed_service_fee"`              // aCAM

	ResponseTimeout int64 `mapstructure:"response_timeout"` // milliseconds

	RecordExpiration bool `mapstructure:"record_expiration"`

	PartnerPlugin PartnerPluginConfig `mapstructure:"partner_plugin"`
	RPCServer     RPCServerConfig     `mapstructure:"rpc_server"`

	DB     UnparsedSQLiteDBConfig `mapstructure:"db"`
	Matrix UnparsedMatrixConfig   `mapstructure:"matrix"`
	CashIn UnparsedCashInConfig   `mapstructure:"cash_in"` // TODO@ keep naming consistent with other cash-in config structs and flags
}

type UnparsedSQLiteDBConfig struct {
	DBPath string `mapstructure:"path"`
}

type UnparsedMatrixConfig struct {
	Host string `mapstructure:"host"`
}

type UnparsedCashInConfig struct { // TODO@ keep naming consistent with other cash-in config structs and flags
	Period    int64  `mapstructure:"period"`     // seconds
	MinAmount uint64 `mapstructure:"min_amount"` // aCAM // TODO@ clarify if its aCAM;
}

func (cfg *Config) unparse() *UnparsedConfig {
	return &UnparsedConfig{
		DB:            cfg.DB.Common,
		RPCServer:     cfg.RPCServer,
		PartnerPlugin: cfg.PartnerPlugin,
		Matrix: UnparsedMatrixConfig{
			Host: cfg.Matrix.Host,
		},
		DeveloperMode:                       cfg.DeveloperMode,
		E2ETestMode:                         cfg.E2ETestMode,
		BotKey:                              hex.EncodeToString(crypto.FromECDSA(cfg.BotKey)),
		CMAccountAddress:                    cfg.CMAccountAddress.Hex(),
		ChainRPCURL:                         cfg.ChainRPCURL,
		BookingTokenAddress:                 cfg.BookingTokenAddress.Hex(),
		NetworkFeeRecipientBotAddress:       cfg.NetworkFeeRecipientBotAddress.Hex(),
		NetworkFeeRecipientCMAccountAddress: cfg.NetworkFeeRecipientCMAccountAddress.Hex(),
		ChequeExpirationTime:                cfg.ChequeExpirationTime.Uint64(),
		MinChequeDurationUntilExpiration:    cfg.MinChequeDurationUntilExpiration.Uint64(),
		CashIn: UnparsedCashInConfig{
			Period:    int64(cfg.CashIn.Period / time.Second),
			MinAmount: cfg.CashIn.MinAmount,
		},
		MaxAllowedServiceFee: cfg.MaxAllowedServiceFee.String(),
		ResponseTimeout:      int64(cfg.ResponseTimeout / time.Millisecond),
		RecordExpiration:     cfg.RecordExpiration,
	}
}
