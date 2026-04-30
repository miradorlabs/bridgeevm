package bridgeevm

import (
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

//go:embed config/*/*.json
var bridgeConfigFS embed.FS

var (
	loadOnce       sync.Once
	loadErr        error
	configsByChain map[string][]*bridgeConfig
)

type bridgeConfig struct {
	ChainName         string          `json:"chainName"`
	BridgeName        string          `json:"bridgeName"`
	BridgeDescription string          `json:"bridgeDescription"`
	BridgeContract    bridgeContract  `json:"bridgeContract"`
	BridgeTopic       bridgeTopicInfo `json:"bridgeTopic"`
}

type bridgeContract struct {
	Address string `json:"address"`
}

type bridgeTopicInfo struct {
	Type        string             `json:"type"`
	Hash        string             `json:"hash"`
	Name        string             `json:"name"`
	Correlation []correlationField `json:"correlation"`
}

// correlationField describes how to extract one component of a bridge's
// correlation ID. After validateBridgeConfig runs, Source and Type are
// guaranteed to be lowercased and trimmed, so the hot path in extraction.go
// can compare them directly without further normalization.
type correlationField struct {
	Index      int    `json:"index"`
	Offset     int    `json:"offset"`
	Size       int    `json:"size"`
	BytesIndex int    `json:"bytesIndex"`
	Type       string `json:"type"`
	Field      string `json:"field"`
	Source     string `json:"source"`
}

func configsForChain(chain string) ([]*bridgeConfig, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	return configsByChain[strings.ToLower(chain)], nil
}

func ensureLoaded() error {
	loadOnce.Do(func() {
		configsByChain, loadErr = loadBridgeConfigs(bridgeConfigFS)
	})
	return loadErr
}

// subscriptionKey identifies a unique (chain, address, topic) tuple. It is
// used at load time to detect collisions across configs; two bridges that
// emit the same event from the same contract on the same chain would
// otherwise silently overwrite each other in the lookup map.
type subscriptionKey struct {
	chain   string
	address common.Address
	topic   common.Hash
}

// loadBridgeConfigs walks configFS for `config/<chain>/*.json` files,
// validates each entry, and returns the configs grouped by lowercased chain
// name. The fs.FS abstraction is used so tests can inject malformed configs
// via fstest.MapFS without round-tripping through the filesystem.
func loadBridgeConfigs(configFS fs.FS) (map[string][]*bridgeConfig, error) {
	chainDirs, err := fs.ReadDir(configFS, "config")
	if err != nil {
		return nil, fmt.Errorf("read embedded bridge config root: %w", err)
	}

	out := make(map[string][]*bridgeConfig)
	seen := make(map[subscriptionKey]string) // value: "filename[idx]" of the prior registration

	for _, chainDir := range chainDirs {
		if !chainDir.IsDir() {
			continue
		}
		chainName := chainDir.Name()

		entries, err := fs.ReadDir(configFS, path.Join("config", chainName))
		if err != nil {
			return nil, fmt.Errorf("read chain dir %s: %w", chainName, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			filePath := path.Join("config", chainName, entry.Name())
			raw, err := fs.ReadFile(configFS, filePath)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", filePath, err)
			}

			var fileConfigs []*bridgeConfig
			if err := json.Unmarshal(raw, &fileConfigs); err != nil {
				return nil, fmt.Errorf("decode %s: %w", filePath, err)
			}

			for idx, cfg := range fileConfigs {
				if err := validateBridgeConfig(filePath, idx, cfg); err != nil {
					return nil, err
				}
				if !strings.EqualFold(cfg.ChainName, chainName) {
					return nil, fmt.Errorf("bridge config %s[%d] chainName %q does not match directory %q",
						filePath, idx, cfg.ChainName, chainName)
				}

				key := subscriptionKey{
					chain:   strings.ToLower(cfg.ChainName),
					address: common.HexToAddress(cfg.BridgeContract.Address),
					topic:   common.HexToHash(cfg.BridgeTopic.Hash),
				}
				if prior, dup := seen[key]; dup {
					return nil, fmt.Errorf(
						"duplicate subscription on %s: %s and %s[%d] both register (%s, %s)",
						key.chain, prior, filePath, idx,
						key.address.Hex(), key.topic.Hex(),
					)
				}
				seen[key] = fmt.Sprintf("%s[%d]", filePath, idx)

				out[key.chain] = append(out[key.chain], cfg)
			}
		}
	}

	return out, nil
}

func validateBridgeConfig(filename string, idx int, cfg *bridgeConfig) error {
	if cfg == nil {
		return fmt.Errorf("nil bridge config in %s index %d", filename, idx)
	}

	cfg.ChainName = strings.TrimSpace(cfg.ChainName)
	cfg.BridgeName = strings.TrimSpace(cfg.BridgeName)
	cfg.BridgeDescription = strings.TrimSpace(cfg.BridgeDescription)

	if cfg.ChainName == "" {
		return fmt.Errorf("bridge config %s[%d] missing chainName", filename, idx)
	}
	if cfg.BridgeName == "" {
		return fmt.Errorf("bridge config %s[%d] missing bridgeName", filename, idx)
	}
	if cfg.BridgeDescription == "" {
		return fmt.Errorf("bridge config %s[%d] missing bridgeDescription", filename, idx)
	}
	if !common.IsHexAddress(cfg.BridgeContract.Address) {
		return fmt.Errorf("bridge config %s[%d] invalid contract address %q", filename, idx, cfg.BridgeContract.Address)
	}
	if err := ensureValidHash(cfg.BridgeTopic.Hash); err != nil {
		return fmt.Errorf("bridge config %s[%d] invalid topic hash: %w", filename, idx, err)
	}

	cfg.BridgeTopic.Type = strings.ToLower(strings.TrimSpace(cfg.BridgeTopic.Type))
	switch cfg.BridgeTopic.Type {
	case "source", "destination":
	case "":
		return fmt.Errorf("bridge config %s[%d] missing bridgeTopic.type", filename, idx)
	default:
		return fmt.Errorf("bridge config %s[%d] invalid bridgeTopic.type %q (must be source or destination)",
			filename, idx, cfg.BridgeTopic.Type)
	}

	if len(cfg.BridgeTopic.Correlation) == 0 {
		return fmt.Errorf("bridge config %s[%d] missing correlation fields", filename, idx)
	}
	for i := range cfg.BridgeTopic.Correlation {
		if err := normalizeAndValidateCorrelationField(filename, idx, i, &cfg.BridgeTopic.Correlation[i]); err != nil {
			return err
		}
	}
	return nil
}

func normalizeAndValidateCorrelationField(filename string, cfgIdx, fieldIdx int, field *correlationField) error {
	field.Source = strings.ToLower(strings.TrimSpace(field.Source))
	if field.Source == "" {
		field.Source = sourceTopics
	}
	field.Type = strings.ToLower(strings.TrimSpace(field.Type))

	switch field.Source {
	case sourceTopics, sourceData:
		if field.Index < 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d index must be >= 0", filename, cfgIdx, fieldIdx)
		}
	case sourcePacked:
		if field.Offset < 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d offset must be >= 0", filename, cfgIdx, fieldIdx)
		}
		if field.Size <= 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d size must be > 0", filename, cfgIdx, fieldIdx)
		}
	case sourceAbiBytes:
		if field.BytesIndex < 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d bytesIndex must be >= 0", filename, cfgIdx, fieldIdx)
		}
		if field.Offset < 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d offset must be >= 0", filename, cfgIdx, fieldIdx)
		}
		if field.Size <= 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d size must be > 0", filename, cfgIdx, fieldIdx)
		}
	case sourceAbiBytesHash:
		if field.BytesIndex < 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d bytesIndex must be >= 0", filename, cfgIdx, fieldIdx)
		}
		if field.Offset < 0 {
			return fmt.Errorf("bridge config %s[%d] correlation field %d offset must be >= 0", filename, cfgIdx, fieldIdx)
		}
	default:
		return fmt.Errorf("bridge config %s[%d] correlation field %d invalid source %q (must be %s, %s, %s, %s, or %s)",
			filename, cfgIdx, fieldIdx, field.Source,
			sourceTopics, sourceData, sourcePacked, sourceAbiBytes, sourceAbiBytesHash)
	}

	if field.Type == "" {
		return fmt.Errorf("bridge config %s[%d] correlation field %d missing type", filename, cfgIdx, fieldIdx)
	}
	// decodeCorrelationValue is the canonical list of supported ABI types.
	// Asking it to render a zero hash exercises the same switch the hot path
	// will run later, so adding a new type to extraction.go automatically
	// makes it acceptable to validation.
	if _, err := decodeCorrelationValue(common.Hash{}, field.Type); err != nil {
		return fmt.Errorf("bridge config %s[%d] correlation field %d: %w", filename, cfgIdx, fieldIdx, err)
	}
	return nil
}

func ensureValidHash(value string) error {
	if !strings.HasPrefix(value, "0x") {
		return fmt.Errorf("value %q missing 0x prefix", value)
	}
	bytes, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return fmt.Errorf("value %q is not hex: %w", value, err)
	}
	if len(bytes) != common.HashLength {
		return fmt.Errorf("value %q is %d bytes; expected %d", value, len(bytes), common.HashLength)
	}
	return nil
}
