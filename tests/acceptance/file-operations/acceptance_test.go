package file_operations

import (
	"os"
	"testing"

	"github.com/sapslaj/mid/tests/acceptance/integration"
	"github.com/sapslaj/mid/tests/must"
)

func TestFileOperations(t *testing.T) {
	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Dir: must.Must1(os.Getwd()),
		Dependencies: []string{
			"@sapslaj/pulumi-mid",
		},
	})
}
