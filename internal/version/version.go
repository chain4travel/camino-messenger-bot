// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"fmt"
	"runtime/debug"
)

var (
	// AppVersion is set by go build -ldflags
	AppVersion = "Unspecified"

	// AppGitCommit is set by go build -ldflags
	AppGitCommit = "Unspecified"

	// BufBuildPBCMPRelease is set by go build -ldflags
	BufBuildPBCMPRelease = "Unspecified"

	// BufBuildGRPCCMPRelease is set by go build -ldflags
	BufBuildGRPCCMPRelease = "Unspecified"

	// BufBuildPBCommit set during init from pkg dependency version
	BufBuildPBCommit = "Unspecified"

	// BufBuildGRPCCommit set during init from pkg dependency version
	BufBuildGRPCCommit = "Unspecified"

	// ContractsGitCommit set during init from pkg dependency version
	ContractsGitCommit = "Unspecified"

	// FullVersion is set during init by combining all version info
	FullVersion = "Unspecified"
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
