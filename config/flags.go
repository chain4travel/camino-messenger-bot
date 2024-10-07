package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagKeyConfig = "config"

	FlagKeyDeveloperMode = "developer_mode"
)

func BindFlags(cmd *cobra.Command) error {
	cmd.PersistentFlags().String(flagKeyConfig, "camino-messenger-bot", "path to config file dir")

	// Main config flags
	cmd.PersistentFlags().Bool(FlagKeyDeveloperMode, false, "Sets developer mode.")
	cmd.PersistentFlags().String("evm_private_key", "", "Sets bot private key. Its used for the matrix server connection, cm account interaction and cheques signing.")
	cmd.PersistentFlags().String("cm_account_address", "", "Sets bot cm account address.")
	cmd.PersistentFlags().String("rpc_url", "", "C-chain RPC URL.")
	cmd.PersistentFlags().String("booking_token_address", "0xe55E387F5474a012D1b048155E25ea78C7DBfBBC", "BookingToken address.")
	cmd.PersistentFlags().String("network_fee_recipient_bot_address", "", "Network fee recipient bot address.")
	cmd.PersistentFlags().String("network_fee_recipient_cm_account", "", "Network fee recipient CMAccount address.")
	cmd.PersistentFlags().Uint64("cheque_expiration_time", 3600*24*30*7, "Cheque expiration time (in seconds).")
	cmd.PersistentFlags().Uint64("min_cheque_duration_until_expiration", 3600*24*30*6, "Minimum valid duration until cheque expiration (in seconds).")
	cmd.PersistentFlags().Uint64("cash_in_period", 3600*24, "Cash-in period (in seconds).")
	cmd.PersistentFlags().Int64("response_timeout", 3000, "The messenger timeout (in milliseconds).")

	// DB config flags
	cmd.PersistentFlags().String("db_name", "camino_messenger_bot", "Database name.")
	cmd.PersistentFlags().String("db_path", "db.db", "Path to database.")
	cmd.PersistentFlags().String("migrations_path", "file://./migrations", "Path to migration scripts.")

	// Tracing config flags
	cmd.PersistentFlags().Bool("tracing_enabled", false, "Whether tracing is enabled.")
	cmd.PersistentFlags().String("tracing_host", "localhost:4317", "The tracing host.")
	cmd.PersistentFlags().Bool("tracing_insecure", true, "Whether the tracing connection should be insecure.")
	cmd.PersistentFlags().String("tracing_cert_file", "", "The tracing certificate file.")
	cmd.PersistentFlags().String("tracing_key_file", "", "The tracing key file.")

	// Partner plugin config flags
	cmd.PersistentFlags().String("partner_plugin_host", "localhost:50051", "partner plugin RPC server host.")
	cmd.PersistentFlags().Bool("partner_plugin_unencrypted", false, "Whether the RPC client should initiate an unencrypted connection with the server.")
	cmd.PersistentFlags().String("partner_plugin_ca_file", "", "The partner plugin RPC server CA certificate file.")

	// RPC server config flags
	cmd.PersistentFlags().Int64("rpc_server_port", 9090, "The RPC server port.")
	cmd.PersistentFlags().Bool("rpc_unencrypted", false, "Whether the RPC server should be unencrypted.")
	cmd.PersistentFlags().String("rpc_server_cert_file", "", "The server certificate file.")
	cmd.PersistentFlags().String("rpc_server_key_file", "", "The server key file.")

	// Matrix config flags
	cmd.PersistentFlags().String("matrix_host", "", "Sets the matrix host.")
	cmd.PersistentFlags().String("matrix_store", "", "Sets the matrix store (sqlite3 db path).")

	return viper.BindPFlags(cmd.PersistentFlags())
}
