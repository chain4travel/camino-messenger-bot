package config

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
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

	errInvalidRawConfig = errors.New("invalid raw config")
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
		cr.logger.Errorf("Error binding flags: %s", err)
		return nil, err
	}

	configPath := cr.viper.GetString(flagKeyConfig)
	if configPath == "" {
		err := errors.New("config path is empty")
		cr.logger.Error(err)
		return nil, err
	}
	cr.viper.SetConfigFile(configPath)

	if err := cr.viper.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			cr.logger.Errorf("Error reading config file: %s", err)
			return nil, err
		}
		cr.logger.Info("Config file not found")
	}

	cfg := &UnparsedConfig{}
	if err := cr.viper.Unmarshal(cfg); err != nil {
		cr.logger.Error(err)
		return nil, err
	}

	parsedCfg, err := cr.parseConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidRawConfig, err)
	}

	return parsedCfg, nil
}

func (cr *reader) parseConfig(cfg *UnparsedConfig) (*Config, error) {
	botKey, err := crypto.HexToECDSA(cfg.BotKey)
	if err != nil {
		cr.logger.Errorf("Error parsing bot key: %s", err)
		return nil, err
	}

	chainRPC, err := url.Parse(cfg.ChainRPCURL)
	if err != nil {
		cr.logger.Errorf("Error parsing C-Chain RPC URL: %s", err)
		return nil, err
	}

	tracingHost, err := url.Parse(cfg.Tracing.Host)
	if err != nil {
		cr.logger.Errorf("Error parsing tracing host: %s", err)
		return nil, err
	}

	partnerPluginHost, err := url.Parse(cfg.PartnerPlugin.Host)
	if err != nil {
		cr.logger.Errorf("Error parsing partner plugin host: %s", err)
		return nil, err
	}

	matrixHost, err := url.Parse(cfg.Matrix.Host)
	if err != nil {
		cr.logger.Errorf("Error parsing matrix host: %s", err)
		return nil, err
	}

	return &Config{
		DB: SQLiteDBConfig{
			Common: cfg.DB,
			Scheduler: UnparsedSQLiteDBConfig{
				DBPath:         cfg.DB.DBPath + "/scheduler",
				MigrationsPath: cfg.DB.MigrationsPath + "/scheduler",
			},
			ChequeHandler: UnparsedSQLiteDBConfig{
				DBPath:         cfg.DB.DBPath + "/cheque_handler",
				MigrationsPath: cfg.DB.MigrationsPath + "/cheque_handler",
			},
		},
		RPCServer: cfg.RPCServer,
		Tracing: TracingConfig{
			Enabled:  cfg.Tracing.Enabled,
			HostURL:  *tracingHost,
			Insecure: cfg.Tracing.Insecure,
			CertFile: cfg.Tracing.CertFile,
			KeyFile:  cfg.Tracing.KeyFile,
		},
		PartnerPlugin: PartnerPluginConfig{
			Enabled:     cfg.PartnerPlugin.Enabled,
			HostURL:     *partnerPluginHost,
			Unencrypted: cfg.PartnerPlugin.Unencrypted,
			CACertFile:  cfg.PartnerPlugin.CACertFile,
		},
		Matrix: MatrixConfig{
			HostURL: *matrixHost,
			Store:   cfg.Matrix.Store,
		},
		DeveloperMode:                       cfg.DeveloperMode,
		BotKey:                              botKey,
		CMAccountAddress:                    common.HexToAddress(cfg.CMAccountAddress),
		ChainRPCURL:                         *chainRPC,
		BookingTokenAddress:                 common.HexToAddress(cfg.BookingTokenAddress),
		NetworkFeeRecipientBotAddress:       common.HexToAddress(cfg.NetworkFeeRecipientBotAddress),
		NetworkFeeRecipientCMAccountAddress: common.HexToAddress(cfg.NetworkFeeRecipientCMAccountAddress),
		ChequeExpirationTime:                big.NewInt(0).SetUint64(cfg.ChequeExpirationTime),
		MinChequeDurationUntilExpiration:    big.NewInt(0).SetUint64(cfg.MinChequeDurationUntilExpiration),
		CashInPeriod:                        time.Duration(cfg.CashInPeriod) * time.Second,
		ResponseTimeout:                     time.Duration(cfg.ResponseTimeout) * time.Millisecond,
	}, nil
}
