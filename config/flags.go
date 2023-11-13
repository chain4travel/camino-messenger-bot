package config

import "flag"

func readAppConfig(cfg AppConfig, fs *flag.FlagSet) {
	fs.BoolVar(&cfg.DeveloperMode, DeveloperMode, false, "Sets developer mode")

}

func readMatrixConfig(cfg MatrixConfig, fs *flag.FlagSet) {
	fs.StringVar(&cfg.Username, MatrixUsername, "", "Sets username used for the matrix server connection")
	fs.StringVar(&cfg.Password, MatrixPassword, "", "Sets password used for the matrix server connection")
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
	fs.StringVar(&cfg.PartnerPluginHost, PartnerPluginHostKey, "", "The partner plugin RPC server host")
	fs.IntVar(&cfg.PartnerPluginPort, PartnerPluginPortKey, 50051, "The partner plugin RPC server port")
}

func readMessengerConfig(cfg ProcessorConfig, fs *flag.FlagSet) {
	fs.IntVar(&cfg.Timeout, MessengerTimeoutKey, 3000, "The messenger timeout (in milliseconds)")
	flag.Var(&cfg.SupportedRequestTypes, SupportedRequestTypesKey, "The list of supported request types")
	flag.Parse()
}
