package config

import (
	"errors"
	"math/big"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const envPrefix = "CMB"

func ReadConfig(logger *zap.SugaredLogger) (*Config, error) {
	cr := &configReader{
		logger: logger,
	}

	return cr.readConfig()
}

type configReader struct {
	logger *zap.SugaredLogger
}

func (cr *configReader) readConfig() (*Config, error) {
	configPath := viper.GetString(flagKeyConfig)
	if configPath == "" {
		err := errors.New("config path is empty")
		cr.logger.Error(err)
		return nil, err
	}

	viper.AddConfigPath(configPath)

	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		viperErr := &viper.ConfigFileNotFoundError{}
		if ok := errors.As(err, viperErr); !ok {
			cr.logger.Errorf("Error reading config file: %s", err)
			return nil, err
		}
		cr.logger.Info("Config file not found")
	}

	cfg := &UnparsedConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		cr.logger.Error(err)
		return nil, err
	}

	return cr.parseConfig(cfg)
}

func (cr *configReader) parseConfig(cfg *UnparsedConfig) (*Config, error) {
	botKey, err := crypto.HexToECDSA(cfg.BotKey)
	if err != nil {
		cr.logger.Errorf("Error parsing bot key: %s", err)
		return nil, err
	}

	cChainRPCURL, err := url.Parse(cfg.CChainRPCURL)
	if err != nil {
		cr.logger.Errorf("Error parsing C-Chain RPC URL: %s", err)
		return nil, err
	}

	var tracingHost *url.URL
	if cfg.UnparsedTracingConfig.Host != "" {
		tracingHost, err = url.Parse(cfg.UnparsedTracingConfig.Host)
		if err != nil {
			cr.logger.Errorf("Error parsing tracing host: %s", err)
			return nil, err
		}
	}

	var partnerPluginHost *url.URL
	if cfg.UnparsedPartnerPluginConfig.Host != "" {
		partnerPluginHost, err = url.Parse(cfg.UnparsedPartnerPluginConfig.Host)
		if err != nil {
			cr.logger.Errorf("Error parsing partner plugin host: %s", err)
			return nil, err
		}
	}

	return &Config{
		MatrixConfig:    cfg.MatrixConfig,
		DBConfig:        cfg.DBConfig,
		RPCServerConfig: cfg.RPCServerConfig,
		TracingConfig: TracingConfig{
			Enabled:  cfg.UnparsedTracingConfig.Enabled,
			HostURL:  tracingHost,
			Insecure: cfg.UnparsedTracingConfig.Insecure,
			CertFile: cfg.UnparsedTracingConfig.CertFile,
			KeyFile:  cfg.UnparsedTracingConfig.KeyFile,
		},
		PartnerPluginConfig: PartnerPluginConfig{
			HostURL:     partnerPluginHost,
			Unencrypted: cfg.UnparsedPartnerPluginConfig.Unencrypted,
			CACertFile:  cfg.UnparsedPartnerPluginConfig.CACertFile,
		},
		DeveloperMode:                       cfg.DeveloperMode,
		BotKey:                              botKey,
		CMAccountAddress:                    common.HexToAddress(cfg.CMAccountAddress),
		CChainRPCURL:                        *cChainRPCURL,
		BookingTokenAddress:                 common.HexToAddress(cfg.BookingTokenAddress),
		NetworkFeeRecipientBotAddress:       common.HexToAddress(cfg.NetworkFeeRecipientBotAddress),
		NetworkFeeRecipientCMAccountAddress: common.HexToAddress(cfg.NetworkFeeRecipientCMAccountAddress),
		ChequeExpirationTime:                big.NewInt(0).SetUint64(cfg.ChequeExpirationTime),
		MinChequeDurationUntilExpiration:    big.NewInt(0).SetUint64(cfg.MinChequeDurationUntilExpiration),
		CashInPeriod:                        time.Duration(cfg.CashInPeriod) * time.Second,
		ResponseTimeout:                     time.Duration(cfg.ResponseTimeout) * time.Millisecond,
	}, nil
}
