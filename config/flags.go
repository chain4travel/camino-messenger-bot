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
	fs.StringVar(&cfg.NodeURI, EvmNodeURIKey, "", "The EVM node URI")
	fs.StringVar(&cfg.PrivateKey, EvmPrivateKey, "", "The EVM private key")
	fs.UintVar(&cfg.NetworkID, EvmNetworkIDKey, 0, "The EVM network ID")
	fs.StringVar(&cfg.ChainID, EvmChainIDKey, "", "The EVM chain ID")
	fs.UintVar(&cfg.AwaitTxConfirmationTimeout, EvmAwaitTxConfirmationTimeout, 3000, "The EVM await transaction confirmation timeout (in milliseconds)")
	fs.StringVar(&cfg.RPCURL, RPCURLKey, "", "The EVM RPC URL")
	fs.StringVar(&cfg.BookingTokenAddress, BookingTokenAddressKey, "0xd4e2D76E656b5060F6f43317E8d89ea81eb5fF8D", "BookingToken address")
	fs.StringVar(&cfg.BookingTokenABIFile, BookingTokenABIFileKey, "./abi/BookingTokenV0.abi", "BookingToken ABI file")
}

func readTracingConfig(cfg TracingConfig, fs *flag.FlagSet) {
	fs.BoolVar(&cfg.Enabled, TracingEnabledKey, false, "Whether tracing is enabled")
	fs.StringVar(&cfg.Host, TracingHostKey, "localhost", "The tracing host")
	fs.IntVar(&cfg.Port, TracingPortKey, 4317, "The tracing port")
	fs.BoolVar(&cfg.Insecure, TracingInsecureKey, true, "Whether the tracing connection should be insecure")
	fs.StringVar(&cfg.CertFile, TracingCertFileKey, "", "The tracing certificate file")
	fs.StringVar(&cfg.KeyFile, TracingKeyFileKey, "", "The tracing key file")
}
