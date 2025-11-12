// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	"go.uber.org/zap"
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

var (
	VersionV1 *typesv1.Version
	VersionV4 *typesv4.Version
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

func InitProtoVersion(logger *zap.SugaredLogger, allowInvalid bool) error {
	splittedAppVersion := strings.Split(AppVersion, "-")
	if len(splittedAppVersion) < 1 {
		return fmt.Errorf("invalid version format: %q, expected format is vMAJOR.MINOR.PATCH", AppVersion)
	}
	splittedAppVersion = strings.Split(strings.TrimPrefix(splittedAppVersion[0], "v"), ".")
	if len(splittedAppVersion) != 3 {
		if !allowInvalid {
			err := fmt.Errorf("invalid version format: %q, expected format is vMAJOR.MINOR.PATCH", AppVersion)
			logger.Error(err)
			return err
		}
		logger.Warnf("invalid version format: %q, using default version 0.0.0 for message headers", AppVersion)
		splittedAppVersion = []string{"0", "0", "0"}
	}

	appVersionMajor, err := strconv.Atoi(splittedAppVersion[0])
	if err != nil {
		return fmt.Errorf("invalid version format: %q, major version must be an integer", AppVersion)
	}
	appVersionMinor, err := strconv.Atoi(splittedAppVersion[1])
	if err != nil {
		return fmt.Errorf("invalid version format: %q, minor version must be an integer", AppVersion)
	}
	appVersionPatch, err := strconv.Atoi(splittedAppVersion[2])
	if err != nil {
		return fmt.Errorf("invalid version format: %q, patch version must be an integer", AppVersion)
	}

	appVersionMajorInt32, err := conversion.IntToInt32(appVersionMajor)
	if err != nil {
		return fmt.Errorf("invalid version format: %q, major version %d exceeds int32 limit", AppVersion, appVersionMajor)
	}
	appVersionMinorInt32, err := conversion.IntToInt32(appVersionMinor)
	if err != nil {
		return fmt.Errorf("invalid version format: %q, minor version %d exceeds int32 limit", AppVersion, appVersionMinor)
	}
	appVersionPatchInt32, err := conversion.IntToInt32(appVersionPatch)
	if err != nil {
		return fmt.Errorf("invalid version format: %q, patch version %d exceeds int32 limit", AppVersion, appVersionPatch)
	}

	appVersionMajorUInt32, err := conversion.IntToUInt32(appVersionMajor)
	if err != nil {
		return fmt.Errorf("invalid version format: %q, major version %d exceeds int32 limit", AppVersion, appVersionMajor)
	}
	appVersionMinorUInt32, err := conversion.IntToUInt32(appVersionMinor)
	if err != nil {
		return fmt.Errorf("invalid version format: %q, minor version %d exceeds int32 limit", AppVersion, appVersionMinor)
	}
	appVersionPatchUInt32, err := conversion.IntToUInt32(appVersionPatch)
	if err != nil {
		return fmt.Errorf("invalid version format: %q, patch version %d exceeds int32 limit", AppVersion, appVersionPatch)
	}

	VersionV1 = &typesv1.Version{
		Major: appVersionMajorInt32,
		Minor: appVersionMinorInt32,
		Patch: appVersionPatchInt32,
	}

	VersionV4 = &typesv4.Version{
		Major: appVersionMajorUInt32,
		Minor: appVersionMinorUInt32,
		Patch: appVersionPatchUInt32,
	}
	return nil
}
