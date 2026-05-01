package bridgeevm

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const correlationDelimiter = ":"

// Correlation source identifiers — the canonical (lowercased) values for
// correlationField.Source. Used by both config validation and the runtime
// dispatch switch so the two cannot drift.
const (
	sourceTopics       = "topics"
	sourceData         = "data"
	sourcePacked       = "packed"
	sourceAbiBytes     = "abi_bytes"
	sourceAbiBytesHash = "abi_bytes_hash"
)

func extractCorrelationFields(log *types.Log, fields []correlationField) (string, error) {
	if len(fields) == 0 {
		return "", fmt.Errorf("no correlation fields configured")
	}

	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value, err := extractFieldValue(log, field)
		if err != nil {
			return "", fmt.Errorf("extract field %q: %w", field.Field, err)
		}
		parts = append(parts, value)
	}

	return strings.Join(parts, correlationDelimiter), nil
}

// extractFieldValue assumes field.Source and field.Type have already been
// normalized (lowercased, trimmed) by normalizeAndValidateCorrelationField at
// config-load time.
func extractFieldValue(log *types.Log, field correlationField) (string, error) {
	switch field.Source {
	case sourceTopics:
		return extractFromTopics(log, field)
	case sourceData:
		return extractFromData(log, field)
	case sourcePacked:
		return extractFromPacked(log, field)
	case sourceAbiBytes:
		return extractFromAbiBytes(log, field)
	case sourceAbiBytesHash:
		return extractFromAbiBytesHash(log, field)
	default:
		return "", fmt.Errorf("unsupported source %q", field.Source)
	}
}

func extractFromTopics(log *types.Log, field correlationField) (string, error) {
	if field.Index >= len(log.Topics) {
		return "", fmt.Errorf("topic index %d out of range (log has %d topics)", field.Index, len(log.Topics))
	}
	return decodeCorrelationValue(log.Topics[field.Index], field.Type)
}

func extractFromData(log *types.Log, field correlationField) (string, error) {
	const wordSize = 32
	start := field.Index * wordSize
	end := start + wordSize
	if end > len(log.Data) {
		return "", fmt.Errorf("data word index %d out of range (log data has %d bytes, need %d)", field.Index, len(log.Data), end)
	}
	return decodeCorrelationValue(common.BytesToHash(log.Data[start:end]), field.Type)
}

func extractFromPacked(log *types.Log, field correlationField) (string, error) {
	end := field.Offset + field.Size
	if end > len(log.Data) {
		return "", fmt.Errorf("packed offset %d+%d out of range (log data has %d bytes)", field.Offset, field.Size, len(log.Data))
	}
	rawBytes := log.Data[field.Offset:end]
	padded := make([]byte, 32)
	copy(padded[32-len(rawBytes):], rawBytes)
	return decodeCorrelationValue(common.BytesToHash(padded), field.Type)
}

func extractFromAbiBytes(log *types.Log, field correlationField) (string, error) {
	bytesData, err := readAbiBytesParam(log, field.BytesIndex)
	if err != nil {
		return "", err
	}

	end := field.Offset + field.Size
	if end > len(bytesData) {
		return "", fmt.Errorf("packed offset %d+%d out of range (bytes data has %d bytes)",
			field.Offset, field.Size, len(bytesData))
	}
	rawBytes := bytesData[field.Offset:end]

	padded := make([]byte, 32)
	copy(padded[32-len(rawBytes):], rawBytes)
	return decodeCorrelationValue(common.BytesToHash(padded), field.Type)
}

func extractFromAbiBytesHash(log *types.Log, field correlationField) (string, error) {
	bytesData, err := readAbiBytesParam(log, field.BytesIndex)
	if err != nil {
		return "", err
	}
	if field.Offset > len(bytesData) {
		return "", fmt.Errorf("offset %d out of range (bytes data has %d bytes)", field.Offset, len(bytesData))
	}
	return crypto.Keccak256Hash(bytesData[field.Offset:]).Hex(), nil
}

// readAbiBytesParam reads an ABI-encoded `bytes` parameter from log.Data.
// log.Data carries the ABI-encoded tail of the event; bytesIndex is the index
// of the offset word that points to the dynamic bytes payload.
//
// Defense in depth: log.Data is attacker-controllable for any contract that
// emits matching topic[0]. We reject any offset or length that exceeds
// MaxInt32 *before* narrowing to a Go int — checking the big.Int directly
// rather than after Uint64(), which would silently truncate values above
// 2^64. There is no legitimate use case for a single log carrying >2GiB of
// payload.
func readAbiBytesParam(log *types.Log, bytesIndex int) ([]byte, error) {
	const wordSize = 32

	if bytesIndex < 0 {
		return nil, fmt.Errorf("bytes index %d must be >= 0", bytesIndex)
	}

	offsetWord := bytesIndex * wordSize
	if offsetWord+wordSize > len(log.Data) {
		return nil, fmt.Errorf("bytes index %d out of range (log data has %d bytes, need %d)",
			bytesIndex, len(log.Data), offsetWord+wordSize)
	}
	offset, err := readBoundedWord(log.Data[offsetWord : offsetWord+wordSize])
	if err != nil {
		return nil, fmt.Errorf("bytes offset %w", err)
	}
	if offset+wordSize > uint64(len(log.Data)) {
		return nil, fmt.Errorf("bytes offset %d out of range (log data has %d bytes)", offset, len(log.Data))
	}

	length, err := readBoundedWord(log.Data[offset : offset+wordSize])
	if err != nil {
		return nil, fmt.Errorf("bytes length %w", err)
	}

	dataStart := offset + wordSize
	if dataStart+length > uint64(len(log.Data)) {
		return nil, fmt.Errorf("bytes data out of range (start %d, length %d, log data has %d bytes)",
			dataStart, length, len(log.Data))
	}
	return log.Data[dataStart : dataStart+length], nil
}

// readBoundedWord interprets a 32-byte big-endian word as a uint64, rejecting
// any value above MaxInt32. The first 28 bytes must be zero and byte 28 must
// have its MSB clear; this rejects anything larger without narrowing through
// uint64, so values > 2^64 cannot silently truncate past the cap.
func readBoundedWord(word []byte) (uint64, error) {
	for i := 0; i < 28; i++ {
		if word[i] != 0 {
			return 0, fmt.Errorf("exceeds max %d", math.MaxInt32)
		}
	}
	if word[28] > 0x7f {
		return 0, fmt.Errorf("exceeds max %d", math.MaxInt32)
	}
	return uint64(word[28])<<24 | uint64(word[29])<<16 | uint64(word[30])<<8 | uint64(word[31]), nil
}

// decodeCorrelationValue renders a 32-byte word as a string keyed off the ABI
// type declared in the bridge config.
//
// Note on signed types: correlation IDs in the configured bridges are
// conceptually non-negative (depositId, nonce, GUID), so signed types like
// int/int64 are decoded as their unsigned magnitude via big.Int.SetBytes.
// If a future bridge ever uses a genuinely-negative correlation ID, this needs
// to be revisited.
func decodeCorrelationValue(topic common.Hash, typ string) (string, error) {
	switch typ {
	case "int", "uint256", "int256", "uint64", "int64", "uint32", "int32", "uint16", "int16":
		return new(big.Int).SetBytes(topic.Bytes()).String(), nil
	case "address":
		return common.BytesToAddress(topic.Bytes()).Hex(), nil
	case "bytes32":
		return topic.Hex(), nil
	default:
		return "", fmt.Errorf("unsupported correlation type %q", typ)
	}
}
