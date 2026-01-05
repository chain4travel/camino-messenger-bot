// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"github.com/spf13/pflag"
)

const (
	flagKeyConfig = "config"

	flagKeyDeveloperMode = "developer_mode"
)

func Flags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("config", pflag.ExitOnError)

	flags.String(flagKeyConfig, "camino-messenger-bot.yaml", "path to config file")

	// Main config flags
	flags.Bool(flagKeyDeveloperMode, false, "Sets developer mode.")
	flags.Bool("e2e_test_mode", false, "Sets e2e test mode adjusting limits and (expiration-)timeouts. DO NOT USE IN PRODUCTION: This mode will fail to work as the deployed contracts are enforcing the restrictions.")
	flags.String("bot_key", "", "Sets bot private key. Its used for the matrix server connection, cm account interaction and cheques signing.")
	flags.String("cm_account_address", "", "Sets bot cm account address.")
	flags.String("chain_rpc_url", "", "C-chain RPC URL.")
	flags.String("booking_token_address", "0xe55E387F5474a012D1b048155E25ea78C7DBfBBC", "BookingToken address.")
	flags.String("network_fee_recipient_bot_address", "", "Network fee recipient bot address.")
	flags.String("network_fee_recipient_cm_account", "", "Network fee recipient CMAccount address.")
	flags.Uint64("cheque_expiration_time", 3600*24*30*7, "Cheque expiration time (in seconds).")
	flags.Uint64("min_cheque_duration_until_expiration", 3600*24*30*6, "Minimum valid duration until cheque expiration (in seconds).")
	flags.Int64("cash_in_period", 3600*24, "Cash-in period (in seconds).")
	flags.String("max_allowed_service_fee", "1000000000000000000", "Maximum allowed service fee (in aCAM).")
	flags.Int64("response_timeout", 3000, "The messenger timeout (in milliseconds).")

	// DB config flags
	flags.String("db.path", "cmb-db", "Path to database dir.")

	// Partner plugin config flags
	flags.Bool("partner_plugin.enabled", false, "Enable or disable the partner plugin rpc client. It must be enabled if bot's cm account supports at least one service.")
	flags.String("partner_plugin.host", "localhost:50051", "partner plugin RPC server host.")
	flags.Bool("partner_plugin.unencrypted", false, "Whether the RPC client should initiate an unencrypted connection with the server.")
	flags.String("partner_plugin.ca_file", "", "The partner plugin RPC server CA certificate file.")

	// RPC server config flags
	flags.Bool("rpc_server.enabled", false, "Enable or disable RPC server. It must be enabled if bot is expecting to receive RPC requests (e.g. its distributor bot).")
	flags.Uint64("rpc_server.port", 9090, "The RPC server port.")
	flags.Bool("rpc_server.unencrypted", false, "Whether the RPC server should be unencrypted.")
	flags.String("rpc_server.cert_file", "", "The server certificate file.")
	flags.String("rpc_server.key_file", "", "The server key file.")

	// Matrix config flags
	flags.String("matrix.host", "", "Sets the matrix host.")

	// Record expiration config flags
	flags.Bool("record_expiration", true, "Whether to record token expiration on chain.")

	return flags
}
