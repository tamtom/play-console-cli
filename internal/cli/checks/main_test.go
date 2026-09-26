package checks

import (
	"os"
	"testing"

	"github.com/tamtom/play-console-cli/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithIsolatedHome(m))
}
