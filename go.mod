module github.com/chain4travel/camino-messenger-bot

go 1.20

require (
	buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go v1.3.0-20231211091155-5467620e05ed.2
	buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go v1.31.0-20231211091155-5467620e05ed.2
	github.com/ava-labs/avalanchego v1.9.16
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/google/uuid v1.4.0
	github.com/klauspost/compress v1.17.3
	github.com/mattn/go-sqlite3 v1.14.18
	github.com/pierrec/lz4 v2.6.1+incompatible
	github.com/rs/zerolog v1.31.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.16.0
	go.uber.org/zap v1.26.0
	golang.org/x/exp v0.0.0-20220426173459-3bcf042a4bf5
	golang.org/x/sync v0.3.0
	google.golang.org/grpc v1.59.0
)

require (
	github.com/btcsuite/btcd/btcutil v1.1.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/tidwall/gjson v1.17.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	maunium.net/go/maulogger/v2 v2.4.1 // indirect
)

require (
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/golang/protobuf v1.5.3
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/spf13/afero v1.10.0 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230920204549-e6e6cdab5c13 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	maunium.net/go/mautrix v0.15.1
)

replace maunium.net/go/mautrix => ./camino-matrix-go

replace github.com/ava-labs/avalanchego => github.com/chain4travel/caminogo v1.0.0-rc1
