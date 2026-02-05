package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenTest runs a .lt file and compares its output to a .expected file.
func goldenTest(t *testing.T, name string) {
	t.Helper()

	ltPath := filepath.Join("..", "..", "testdata", name+".lt")
	expectedPath := filepath.Join("..", "..", "testdata", name+".expected")

	source, err := os.ReadFile(ltPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", ltPath, err)
	}

	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", expectedPath, err)
	}

	got, err := runSource(string(source))
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	expectedStr := strings.TrimRight(string(expected), "\n")
	gotStr := strings.TrimRight(got, "\n")

	if gotStr != expectedStr {
		expectedLines := strings.Split(expectedStr, "\n")
		gotLines := strings.Split(gotStr, "\n")

		t.Errorf("output mismatch for %s", name)
		maxLines := len(expectedLines)
		if len(gotLines) > maxLines {
			maxLines = len(gotLines)
		}
		for i := 0; i < maxLines; i++ {
			var exp, g string
			if i < len(expectedLines) {
				exp = expectedLines[i]
			} else {
				exp = "<missing>"
			}
			if i < len(gotLines) {
				g = gotLines[i]
			} else {
				g = "<missing>"
			}
			prefix := "  "
			if exp != g {
				prefix = "! "
			}
			t.Logf("%sline %d: expected=%q got=%q", prefix, i+1, exp, g)
		}
	}
}

func TestGoldenArray(t *testing.T) {
	goldenTest(t, "golden_array")
}

func TestGoldenFor(t *testing.T) {
	goldenTest(t, "golden_for")
}

func TestGoldenComplex(t *testing.T) {
	goldenTest(t, "golden_complex")
}

func TestGoldenFeatures(t *testing.T) {
	goldenTest(t, "golden_features")
}
