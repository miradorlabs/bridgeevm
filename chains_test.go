package bridgeevm

import (
	"io/fs"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChainConstantsMatchConfigDirs pins the exported Chain* constants to the
// embedded config/<chain>/ directory set. The constants list is written out
// explicitly (not derived via reflection) so adding a new chain forces a
// deliberate edit in both places: a new config/<chain>/ directory and a
// matching const in chains.go.
func TestChainConstantsMatchConfigDirs(t *testing.T) {
	declared := []string{
		ChainArbitrum,
		ChainBase,
		ChainBSC,
		ChainEthereum,
		ChainOptimism,
		ChainPolygon,
	}
	sort.Strings(declared)

	entries, err := fs.ReadDir(bridgeConfigFS, "config")
	require.NoError(t, err)

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	sort.Strings(dirs)

	assert.Equal(t, declared, dirs,
		"Chain* constants in chains.go must mirror config/ subdirectories exactly")
}
