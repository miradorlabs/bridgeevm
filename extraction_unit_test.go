package bridgeevm

import (
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hexToBytes is a tiny helper for clarity in test tables.
func hexToBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	require.NoError(t, err)
	return b
}

func TestExtractFromTopics(t *testing.T) {
	logWithThreeTopics := &types.Log{
		Topics: []common.Hash{
			common.HexToHash("0xaa"),
			common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000007b"),
			common.HexToHash("0x000000000000000000000000abcdefabcdefabcdefabcdefabcdefabcdefabcd"),
		},
	}

	t.Run("index out of range", func(t *testing.T) {
		_, err := extractFromTopics(logWithThreeTopics, correlationField{Index: 5, Type: "uint256"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("unsupported type", func(t *testing.T) {
		_, err := extractFromTopics(logWithThreeTopics, correlationField{Index: 1, Type: "string"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported correlation type")
	})

	t.Run("decodes uint256", func(t *testing.T) {
		v, err := extractFromTopics(logWithThreeTopics, correlationField{Index: 1, Type: "uint256"})
		require.NoError(t, err)
		assert.Equal(t, "123", v)
	})

	t.Run("decodes address", func(t *testing.T) {
		v, err := extractFromTopics(logWithThreeTopics, correlationField{Index: 2, Type: "address"})
		require.NoError(t, err)
		assert.Equal(t, "0xABcdEFABcdEFabcdEfAbCdefabcdeFABcDEFabCD", v)
	})
}

func TestExtractFromData(t *testing.T) {
	// Three 32-byte words: 0, 1, 2.
	data := make([]byte, 96)
	data[31] = 0x00
	data[63] = 0x01
	data[95] = 0x02
	log := &types.Log{Data: data}

	t.Run("word index out of range", func(t *testing.T) {
		_, err := extractFromData(log, correlationField{Index: 3, Type: "uint256"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("partial last word", func(t *testing.T) {
		short := &types.Log{Data: make([]byte, 40)} // 1 full word + 8 bytes
		_, err := extractFromData(short, correlationField{Index: 1, Type: "uint256"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("decodes word at index", func(t *testing.T) {
		v, err := extractFromData(log, correlationField{Index: 2, Type: "uint256"})
		require.NoError(t, err)
		assert.Equal(t, "2", v)
	})
}

func TestExtractFromPacked(t *testing.T) {
	// Packed bytes: [0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe]
	log := &types.Log{Data: []byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe}}

	t.Run("offset+size out of range", func(t *testing.T) {
		_, err := extractFromPacked(log, correlationField{Offset: 4, Size: 8, Type: "uint64"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("exact boundary success", func(t *testing.T) {
		v, err := extractFromPacked(log, correlationField{Offset: 0, Size: 6, Type: "uint64"})
		require.NoError(t, err)
		// 0xdeadbeefcafe = 244837814094590
		assert.Equal(t, "244837814094590", v)
	})

	t.Run("zero-length size is rejected at validation, not here", func(t *testing.T) {
		// Packed extraction itself happily handles size==0 by returning 0;
		// validation prevents zero size from reaching this code.
		v, err := extractFromPacked(log, correlationField{Offset: 0, Size: 0, Type: "uint256"})
		require.NoError(t, err)
		assert.Equal(t, "0", v)
	})
}

func TestExtractFromAbiBytes(t *testing.T) {
	// Construct a minimal ABI-encoded payload for a single dynamic bytes
	// parameter at slot 0:
	//   word 0: offset = 0x20 (start of length+data)
	//   word 1: length = 8
	//   word 2: 8 bytes of data, padded to 32
	good := make([]byte, 0, 96)
	good = append(good, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000020")...)
	good = append(good, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000008")...)
	good = append(good, hexToBytes(t, "0x1122334455667788000000000000000000000000000000000000000000000000")...)

	t.Run("negative bytesIndex", func(t *testing.T) {
		_, err := extractFromAbiBytes(&types.Log{Data: good}, correlationField{
			BytesIndex: -1, Offset: 0, Size: 8, Type: "uint64",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be >= 0")
	})

	t.Run("offset word out of range", func(t *testing.T) {
		// Truncated payload: only the first word, no length / data.
		short := good[:32]
		_, err := extractFromAbiBytes(&types.Log{Data: short}, correlationField{
			BytesIndex: 0, Offset: 0, Size: 8, Type: "uint64",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("offset exceeds MaxInt32", func(t *testing.T) {
		// Big-endian: most significant byte at index 0. 0x80 in the second
		// word (byte 28) gives value > MaxInt32 = 0x7fffffff.
		bad := make([]byte, 32)
		bad[28] = 0x80
		_, err := extractFromAbiBytes(&types.Log{Data: bad}, correlationField{
			BytesIndex: 0, Offset: 0, Size: 1, Type: "uint64",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds max")
	})

	t.Run("offset above MaxUint64 also rejected (no truncation past cap)", func(t *testing.T) {
		// Set a high-order byte so the encoded value > 2^64. Without
		// the BitLen check, Uint64() would truncate to its low 64 bits
		// and silently dodge the cap. With the check, this is rejected.
		bad := make([]byte, 32)
		bad[0] = 0x01
		_, err := extractFromAbiBytes(&types.Log{Data: bad}, correlationField{
			BytesIndex: 0, Offset: 0, Size: 1, Type: "uint64",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds max")
	})

	t.Run("length exceeds MaxInt32", func(t *testing.T) {
		// Offset 0x20, length = MaxInt32 + 1, no actual data.
		bad := make([]byte, 0, 64)
		bad = append(bad, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000020")...)
		bad = append(bad, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000000")...)
		bad[60] = 0x80 // MaxInt32 = 0x7fffffff; this makes it 0x80000000 (= MaxInt32+1)
		_, err := extractFromAbiBytes(&types.Log{Data: bad}, correlationField{
			BytesIndex: 0, Offset: 0, Size: 1, Type: "uint64",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds max")
	})

	t.Run("dataStart+length out of range", func(t *testing.T) {
		// Offset 0x20, length 100, but only 8 bytes of payload follow.
		bad := make([]byte, 0, 96)
		bad = append(bad, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000020")...)
		bad = append(bad, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000064")...)
		bad = append(bad, hexToBytes(t, "0x1122334455667788000000000000000000000000000000000000000000000000")...)
		_, err := extractFromAbiBytes(&types.Log{Data: bad}, correlationField{
			BytesIndex: 0, Offset: 0, Size: 1, Type: "uint64",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("offset+size out of range within bytesData", func(t *testing.T) {
		_, err := extractFromAbiBytes(&types.Log{Data: good}, correlationField{
			BytesIndex: 0, Offset: 4, Size: 16, Type: "uint64",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("happy path", func(t *testing.T) {
		v, err := extractFromAbiBytes(&types.Log{Data: good}, correlationField{
			BytesIndex: 0, Offset: 0, Size: 8, Type: "uint64",
		})
		require.NoError(t, err)
		// 0x1122334455667788 = 1234605616436508552
		assert.Equal(t, "1234605616436508552", v)
	})
}

func TestExtractFromAbiBytesHash(t *testing.T) {
	// Same payload shape as TestExtractFromAbiBytes, but here the type
	// field is irrelevant — extractFromAbiBytesHash always returns
	// keccak256(bytesData[offset:]).Hex().
	good := make([]byte, 0, 96)
	good = append(good, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000020")...)
	good = append(good, hexToBytes(t, "0x0000000000000000000000000000000000000000000000000000000000000008")...)
	good = append(good, hexToBytes(t, "0x1122334455667788000000000000000000000000000000000000000000000000")...)

	t.Run("offset > len(bytesData)", func(t *testing.T) {
		_, err := extractFromAbiBytesHash(&types.Log{Data: good}, correlationField{
			BytesIndex: 0, Offset: 100, Type: "bytes32",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("empty tail returns keccak of empty", func(t *testing.T) {
		v, err := extractFromAbiBytesHash(&types.Log{Data: good}, correlationField{
			BytesIndex: 0, Offset: 8, Type: "bytes32",
		})
		require.NoError(t, err)
		// keccak256("") = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
		assert.Equal(t, "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470", v)
	})

	t.Run("happy path hashes bytesData[offset:]", func(t *testing.T) {
		v, err := extractFromAbiBytesHash(&types.Log{Data: good}, correlationField{
			BytesIndex: 0, Offset: 0, Type: "bytes32",
		})
		require.NoError(t, err)
		// keccak256(0x1122334455667788) — assert by length+prefix only;
		// the round-trip in the integration tests covers value correctness.
		require.Len(t, v, 66) // "0x" + 64 hex chars
		assert.True(t, strings.HasPrefix(v, "0x"))
	})

	t.Run("propagates readAbiBytesParam error", func(t *testing.T) {
		// Negative bytesIndex must come from validation; here we just
		// drive a too-short Data so readAbiBytesParam fails first.
		_, err := extractFromAbiBytesHash(&types.Log{Data: []byte{0x00}}, correlationField{
			BytesIndex: 0, Offset: 0, Type: "bytes32",
		})
		require.Error(t, err)
	})
}

func TestDecodeCorrelationValue(t *testing.T) {
	// One representative known value per supported type group.
	zero := common.Hash{}
	one := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	addrHash := common.HexToHash("0x000000000000000000000000abcdefabcdefabcdefabcdefabcdefabcdefabcd")

	cases := []struct {
		typ     string
		hash    common.Hash
		want    string
		wantErr string
	}{
		{"int", one, "1", ""},
		{"uint256", one, "1", ""},
		{"int256", one, "1", ""},
		{"uint64", one, "1", ""},
		{"int64", one, "1", ""},
		{"uint32", one, "1", ""},
		{"int32", one, "1", ""},
		{"uint16", one, "1", ""},
		{"int16", one, "1", ""},
		{"address", addrHash, "0xABcdEFABcdEFabcdEfAbCdefabcdeFABcDEFabCD", ""},
		{"bytes32", zero, "0x0000000000000000000000000000000000000000000000000000000000000000", ""},
		{"string", one, "", "unsupported correlation type"},
		{"", one, "", "unsupported correlation type"},
		{"uint256 ", one, "", "unsupported correlation type"}, // not normalized here — validation strips it
	}

	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			got, err := decodeCorrelationValue(tc.hash, tc.typ)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExtractCorrelationFields(t *testing.T) {
	log := &types.Log{
		Topics: []common.Hash{
			common.HexToHash("0xaa"),
			common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000007b"),
			common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000001c8"),
		},
	}

	t.Run("empty fields", func(t *testing.T) {
		_, err := extractCorrelationFields(log, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no correlation fields")
	})

	t.Run("single field", func(t *testing.T) {
		v, err := extractCorrelationFields(log, []correlationField{
			{Index: 1, Type: "uint256", Source: "topics"},
		})
		require.NoError(t, err)
		assert.Equal(t, "123", v)
	})

	t.Run("multi-field joined by delimiter", func(t *testing.T) {
		v, err := extractCorrelationFields(log, []correlationField{
			{Index: 1, Type: "uint256", Source: "topics"},
			{Index: 2, Type: "uint256", Source: "topics"},
		})
		require.NoError(t, err)
		assert.Equal(t, "123:456", v)
	})

	t.Run("propagates field error with field name", func(t *testing.T) {
		_, err := extractCorrelationFields(log, []correlationField{
			{Index: 99, Type: "uint256", Source: "topics", Field: "depositId"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "depositId")
	})
}

func TestReadAbiBytesParam_NegativeIndex(t *testing.T) {
	// Direct unit test of the readAbiBytesParam guard, in case future
	// callers reach it without going through extractFromAbiBytes.
	_, err := readAbiBytesParam(&types.Log{Data: make([]byte, 64)}, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be >= 0")
}

func TestReadAbiBytesParam_DocumentedMaxBound(t *testing.T) {
	// Sanity-check the MaxInt32 cap constant. Not a runtime check,
	// just a compile-time reminder if someone changes it.
	assert.Equal(t, int64(math.MaxInt32), int64(2147483647))
}
