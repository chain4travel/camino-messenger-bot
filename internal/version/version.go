package version

import (
	"fmt"
	"runtime/debug"
)

var (
	// set by go build -ldflags
	AppVersion   = "Unspecified"
	AppGitCommit = "Unspecified"

	BufBuildPBCMPRelease   = "Unspecified"
	BufBuildGRPCCMPRelease = "Unspecified"

	// set during init
	BufBuildPBCommit   = "Unspecified"
	BufBuildGRPCCommit = "Unspecified"
	ContractsGitCommit = "Unspecified"
	FullVersion        = "Unspecified"
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

	FullVersion = fmt.Sprintf("%s (git: %s)\n\nlibs:\n  %s: %s (%s)\n  %s: %s (%s)\n  %s: %s",
		AppVersion,
		AppGitCommit,
		"buf.build protocolbuffers ",
		BufBuildPBCommit,
		BufBuildPBCMPRelease,
		"buf.build grpc            ",
		BufBuildGRPCCommit,
		BufBuildGRPCCMPRelease,
		"camino-messenger-contracts",
		ContractsGitCommit,
	)
}
