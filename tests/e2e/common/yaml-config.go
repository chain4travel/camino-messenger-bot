// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"fmt"
	"os"

	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v3"
)

func WriteYAMLConfig(config any, configPath string) error {
	unparsedMap := &map[string]any{}
	if err := mapstructure.Decode(config, unparsedMap); err != nil {
		return fmt.Errorf("failed to parse config into map: %w", err)
	}
	configBytes, err := yaml.Marshal(unparsedMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config map into yaml: %w", err)
	}

	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
