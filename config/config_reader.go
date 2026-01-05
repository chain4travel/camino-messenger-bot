// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const envPrefix = "CMB"

var (
	_ Reader = (*reader)(nil)

	errInvalidRawConfig                           = errors.New("invalid raw config")
	errEmptyConfigPath                            = errors.New("config path is empty")
	errInvalidCMAccountAddress                    = errors.New("invalid CM account address")
	errInvalidBookingTokenAddress                 = errors.New("invalid booking token address")
	errInvalidNetworkFeeRecipientBotAddress       = errors.New("invalid network fee recipient bot address")
	errInvalidNetworkFeeRecipientCMAccountAddress = errors.New("invalid network fee recipient CM account address")
	errInvalidMaxAllowedServiceFee                = errors.New("invalid max allowed service fee")
)

type Reader interface {
	IsDevelopmentMode() bool
	ReadConfig() (*Config, error)
}

// Returns a new config reader.
func NewConfigReader(flags *pflag.FlagSet, logger *zap.SugaredLogger) (Reader, error) {
	return &reader{
		viper:  viper.New(),
		flags:  flags,
		logger: logger,
	}, nil
}

type reader struct {
	viper  *viper.Viper
	logger *zap.SugaredLogger
	flags  *pflag.FlagSet
}

func (cr *reader) IsDevelopmentMode() bool {
	return cr.viper.GetBool(flagKeyDeveloperMode)
}

func (cr *reader) ReadConfig() (*Config, error) {
	cr.viper.SetEnvPrefix(envPrefix)
	cr.viper.AutomaticEnv()
	cr.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := cr.viper.BindPFlags(cr.flags); err != nil {
		err = fmt.Errorf("failed to bind flags: %w", err)
		cr.logger.Error(err)
		return nil, err
	}

	configPath := cr.viper.GetString(flagKeyConfig)
	if configPath == "" {
		cr.logger.Error(errEmptyConfigPath)
		return nil, errEmptyConfigPath
	}
	cr.viper.SetConfigFile(configPath)

	if err := cr.viper.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			cr.logger.Errorf("Error reading config file: %v", err)
			return nil, err
		}
		cr.logger.Info("Config file not found")
	}

	cfg := &UnparsedConfig{}
	if err := cr.viper.Unmarshal(cfg); err != nil {
		err = fmt.Errorf("failed to unmarshal config: %w", err)
		cr.logger.Error(err)
		return nil, err
	}

	parsedCfg, err := cr.parseConfig(cfg)
	if err != nil {
		err = fmt.Errorf("%w: %w", errInvalidRawConfig, err)
		cr.logger.Error(err)
		return nil, err
	}

	return parsedCfg, nil
}

func (cr *reader) parseConfig(cfg *UnparsedConfig) (*Config, error) {
	botKey, err := crypto.HexToECDSA(cfg.BotKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bot key: %w", err)
	}

	if !common.IsHexAddress(cfg.CMAccountAddress) {
		return nil, errInvalidCMAccountAddress
	}

	if !common.IsHexAddress(cfg.BookingTokenAddress) {
		return nil, errInvalidBookingTokenAddress
	}

	if !common.IsHexAddress(cfg.NetworkFeeRecipientBotAddress) {
		return nil, errInvalidNetworkFeeRecipientBotAddress
	}

	if !common.IsHexAddress(cfg.NetworkFeeRecipientCMAccountAddress) {
		return nil, errInvalidNetworkFeeRecipientCMAccountAddress
	}

	maxAllowedServiceFee, ok := new(big.Int).SetString(cfg.MaxAllowedServiceFee, 10)
	if !ok {
		return nil, errInvalidMaxAllowedServiceFee
	}
	if maxAllowedServiceFee.Sign() < 0 {
		return nil, errInvalidMaxAllowedServiceFee
	}

	return &Config{
		DB: SQLiteDBConfig{
			Common: cfg.DB,
			Scheduler: UnparsedSQLiteDBConfig{
				DBPath: cfg.DB.DBPath + "/scheduler",
			},
			ChequeHandler: UnparsedSQLiteDBConfig{
				DBPath: cfg.DB.DBPath + "/cheque_handler",
			},
			EventListener: UnparsedSQLiteDBConfig{
				DBPath: cfg.DB.DBPath + "/event_listener",
			},
			MessagesEncoderDecoder: UnparsedSQLiteDBConfig{
				DBPath: cfg.DB.DBPath + "/messages_encoder_decoder",
			},
			Resolver: UnparsedSQLiteDBConfig{
				DBPath: cfg.DB.DBPath + "/resolver",
			},
		},
		RPCServer:     cfg.RPCServer,
		PartnerPlugin: cfg.PartnerPlugin,
		Matrix: MatrixConfig{
			Host:  cfg.Matrix.Host,
			Store: cfg.DB.DBPath + "/matrix",
		},
		DeveloperMode:                       cfg.DeveloperMode,
		E2ETestMode:                         cfg.E2ETestMode,
		BotKey:                              botKey,
		CMAccountAddress:                    common.HexToAddress(cfg.CMAccountAddress),
		ChainRPCURL:                         cfg.ChainRPCURL,
		BookingTokenAddress:                 common.HexToAddress(cfg.BookingTokenAddress),
		NetworkFeeRecipientBotAddress:       common.HexToAddress(cfg.NetworkFeeRecipientBotAddress),
		NetworkFeeRecipientCMAccountAddress: common.HexToAddress(cfg.NetworkFeeRecipientCMAccountAddress),
		ChequeExpirationTime:                big.NewInt(0).SetUint64(cfg.ChequeExpirationTime),
		MinChequeDurationUntilExpiration:    big.NewInt(0).SetUint64(cfg.MinChequeDurationUntilExpiration),
		CashInPeriod:                        time.Duration(cfg.CashInPeriod) * time.Second,
		MaxAllowedServiceFee:                maxAllowedServiceFee,
		ResponseTimeout:                     time.Duration(cfg.ResponseTimeout) * time.Millisecond,
		RecordExpiration:                    cfg.RecordExpiration,
	}, nil
}
