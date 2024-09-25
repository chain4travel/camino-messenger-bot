package config

import "flag"

func readAppConfig(cfg AppConfig, fs *flag.FlagSet) {
	fs.BoolVar(&cfg.DeveloperMode, DeveloperMode, false, "Sets developer mode")
	fs.Var(&cfg.SupportedRequestTypes, SupportedRequestTypesKey, "The list of supported request types")
	flag.Parse()
}

func readMatrixConfig(cfg MatrixConfig, fs *flag.FlagSet) {
	fs.StringVar(&cfg.Key, MatrixKey, "", "Sets private key used for the matrix server connection")
	fs.StringVar(&cfg.Host, MatrixHost, "", "Sets the matrix host")
	fs.StringVar(&cfg.Store, MatrixStore, "", "Sets the matrix store (sqlite3 db path)")
}

func readRPCServerConfig(cfg RPCServerConfig, fs *flag.FlagSet) {
	fs.IntVar(&cfg.Port, RPCServerPortKey, 9090, "The RPC server port")
	fs.BoolVar(&cfg.Unencrypted, RPCUnencryptedKey, false, "Whether the RPC server should be unencrypted")
	fs.StringVar(&cfg.ServerCertFile, RPCServerCertFileKey, "", "The server certificate file")
	fs.StringVar(&cfg.ServerKeyFile, RPCServerKeyFileKey, "", "The server key file")
}

func readPartnerRPCServerConfig(cfg PartnerPluginConfig, fs *flag.FlagSet) {
	fs.StringVar(&cfg.Host, PartnerPluginHostKey, "", "The partner plugin RPC server host")
	fs.IntVar(&cfg.Port, PartnerPluginPortKey, 50051, "The partner plugin RPC server port")
	fs.BoolVar(&cfg.Unencrypted, PartnerPluginUnencryptedKey, false, "Whether the RPC client should initiate an unencrypted connection with the server")
	fs.StringVar(&cfg.CACertFile, PartnerPluginCAFileKey, "", "The partner plugin RPC server CA certificate file")
}

func readMessengerConfig(cfg ProcessorConfig, fs *flag.FlagSet) {
	fs.IntVar(&cfg.Timeout, MessengerTimeoutKey, 3000, "The messenger timeout (in milliseconds)")
}

func readEvmConfig(cfg EvmConfig, fs *flag.FlagSet) {
	fs.StringVar(&cfg.PrivateKey, EvmPrivateKey, "", "The EVM private key")
	fs.StringVar(&cfg.RPCURL, RPCURLKey, "", "The EVM RPC URL")
	fs.StringVar(&cfg.BookingTokenAddress, BookingTokenAddressKey, "0xe55E387F5474a012D1b048155E25ea78C7DBfBBC", "BookingToken address")
	fs.Uint64Var(&cfg.BuyableUntilDefault, BuyableUntilDefaultKey, 600, "How log the Token is buyable in seconds")
	fs.StringVar(&cfg.CMAccountAddress, CMAccountAddressKey, "", "CMAccount address")
}

func readTracingConfig(cfg TracingConfig, fs *flag.FlagSet) {
	fs.BoolVar(&cfg.Enabled, TracingEnabledKey, false, "Whether tracing is enabled")
	fs.StringVar(&cfg.Host, TracingHostKey, "localhost", "The tracing host")
	fs.IntVar(&cfg.Port, TracingPortKey, 4317, "The tracing port")
	fs.BoolVar(&cfg.Insecure, TracingInsecureKey, true, "Whether the tracing connection should be insecure")
	fs.StringVar(&cfg.CertFile, TracingCertFileKey, "", "The tracing certificate file")
	fs.StringVar(&cfg.KeyFile, TracingKeyFileKey, "", "The tracing key file")
}

func readDBConfig(cfg DBConfig, fs *flag.FlagSet) {
	fs.StringVar(&cfg.DBName, DBNameKey, "camino_messenger_bot", "database name")
	fs.StringVar(&cfg.DBPath, DBPathKey, "db.db", "path to database")
	fs.StringVar(&cfg.MigrationsPath, MigrationsPathKey, "file://./migrations", "path to migration scripts")
}
