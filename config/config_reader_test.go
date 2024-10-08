package config

import (
	_ "embed"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

//go:embed test_config.yaml
var testConfigYAML []byte

func TestReadConfig(t *testing.T) {
	const nonExistingConfigPath = "non-existing-file.yaml"
	const existingConfigPath = "test_config.yaml"

	var rawMap map[string]interface{}
	require.NoError(t, yaml.Unmarshal(testConfigYAML, &rawMap))

	tests := map[string]struct {
		prepare     func(*testing.T, *configReader)
		flags       *pflag.FlagSet
		expectedErr error
	}{
		"default": {
			prepare: func(_ *testing.T, cr *configReader) {
				cr.viper.Set(flagKeyConfig, nonExistingConfigPath)
			},
			flags:       Flags(),
			expectedErr: errInvalidConfig, // empty bot key
		},
		"from file": {
			prepare: func(_ *testing.T, cr *configReader) {
				cr.viper.Set(flagKeyConfig, existingConfigPath)
			},
			flags: Flags(),
		},
		"from env": {
			prepare: func(t *testing.T, cr *configReader) {
				setEnvFromMap(envPrefix, rawMap)
				cr.viper.Set(flagKeyConfig, nonExistingConfigPath)
			},
			flags: Flags(),
		},
		"from flags": {
			prepare: func(t *testing.T, cr *configReader) {
				setFlagsFromMap(cr.flags, "", rawMap)
				cr.viper.Set(flagKeyConfig, nonExistingConfigPath)
			},
			flags: Flags(),
		},
		// TODO @evlekht add test case for priority of sources
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			configReader := NewConfigReader(tt.flags)
			tt.prepare(t, configReader)

			config, err := configReader.ReadConfig(zap.NewNop().Sugar())
			require.ErrorIs(t, err, tt.expectedErr)

			unsetEnvFromMap(envPrefix, rawMap)

			if err == nil {
				unparsedMap := &map[string]any{}
				require.NoError(t, mapstructure.Decode(config.unparse(), unparsedMap))
				configYAML, err := yaml.Marshal(unparsedMap)
				require.NoError(t, err)
				require.Equal(t, string(testConfigYAML), string(configYAML))
			}
		})
	}
}

func setEnvFromMap(prefix string, m map[string]interface{}) error {
	for key, value := range m {
		fullKey := prefix + "_" + key
		if reflect.TypeOf(value).Kind() == reflect.Map {
			nestedMap, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid nested map type for key %s", fullKey)
			}
			if err := setEnvFromMap(fullKey, nestedMap); err != nil {
				return err
			}
		} else {
			if err := os.Setenv(strings.ToUpper(fullKey), fmt.Sprintf("%v", value)); err != nil {
				return fmt.Errorf("error setting env var for %s: %w", fullKey, err)
			}
		}
	}
	return nil
}

func unsetEnvFromMap(prefix string, m map[string]interface{}) error {
	for key, value := range m {
		fullKey := strings.ToUpper(prefix + "_" + key)
		if reflect.TypeOf(value).Kind() == reflect.Map {
			nestedMap, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid nested map type for key %s", fullKey)
			}
			if err := unsetEnvFromMap(fullKey, nestedMap); err != nil {
				return err
			}
		} else {
			if err := os.Unsetenv(strings.ToUpper(fullKey)); err != nil {
				return fmt.Errorf("error setting env var for %s: %w", fullKey, err)
			}
		}
	}
	return nil
}

func setFlagsFromMap(flagSet *pflag.FlagSet, prefix string, m map[string]interface{}) error {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if reflect.TypeOf(value).Kind() == reflect.Map {
			nestedMap, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid nested map type for key %s", fullKey)
			}
			if err := setFlagsFromMap(flagSet, fullKey, nestedMap); err != nil {
				return err
			}
		} else {
			if err := flagSet.Set(fullKey, fmt.Sprintf("%v", value)); err != nil {
				return fmt.Errorf("error setting flag var for %s: %w", fullKey, err)
			}
		}
	}
	return nil
}
