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

func readPartnerRpcServerConfig(cfg PartnerPluginConfig, fs *flag.FlagSet) {
	fs.StringVar(&cfg.Host, PartnerPluginHostKey, "", "The partner plugin RPC server host")
	fs.IntVar(&cfg.Port, PartnerPluginPortKey, 50051, "The partner plugin RPC server port")
	fs.BoolVar(&cfg.Unencrypted, PartnerPluginUnencryptedKey, false, "Whether the RPC client should initiate an unencrypted connection with the server")
	fs.StringVar(&cfg.CACertFile, PartnerPluginCAFileKey, "", "The partner plugin RPC server CA certificate file")

}

func readMessengerConfig(cfg ProcessorConfig, fs *flag.FlagSet) {
	fs.IntVar(&cfg.Timeout, MessengerTimeoutKey, 3000, "The messenger timeout (in milliseconds)")
}

func readTvmConfig(cfg TvmConfig, fs *flag.FlagSet) {
	fs.StringVar(&cfg.NodeURI, TvmNodeURIKey, "", "The TVM node URI")
	fs.StringVar(&cfg.PrivateKey, TvmPrivateKeyKey, "", "The TVM private key")
	fs.UintVar(&cfg.NetworkID, TvmNetworkIDKey, 0, "The TVM network ID")
	fs.StringVar(&cfg.ChainID, TvmChainIDKey, "", "The TVM chain ID")
}
