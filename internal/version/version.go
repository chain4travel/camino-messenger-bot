package version

import "runtime/debug"

const ProtocolVersion = "v10.0.0"

var (
	// AppVersion is set by go build -ldflags
	AppVersion = "Unspecified"

	// AppGitCommit is set by go build -ldflags
	AppGitCommit = "Unspecified"

	BufBuildPBCommit   = "Unspecified"
	BufBuildGRPCCommit = "Unspecified"
	ContractsGitCommit = "Unspecified"
)

func init() {
	info, _ := debug.ReadBuildInfo()
	for _, dependency := range info.Deps {
		switch dependency.Path {
		case "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go":
			BufBuildPBCommit = dependency.Version
		case "buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go":
			BufBuildGRPCCommit = dependency.Version
		case "github.com/chain4travel/camino-messenger-contracts/go/contracts":
			ContractsGitCommit = dependency.Version
		}
	}
}
