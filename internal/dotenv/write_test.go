package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetUnset(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# keep me\nA=1\nexport B=2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Set(path, map[string]string{"B": "two words", "C": `say "hi"`}); err != nil {
		t.Fatal(err)
	}
	removed, err := Unset(path, []string{"A", "MISSING"})
	if err != nil || len(removed) != 1 || removed[0] != "A" {
		t.Fatalf("Unset = %v, %v", removed, err)
	}
	data, _ := os.ReadFile(path)
	want := "# keep me\nB=\"two words\"\nC=\"say \\\"hi\\\"\"\n"
	if string(data) != want {
		t.Fatalf("file = %q, want %q", data, want)
	}
	vars, _ := Load(path)
	if vars["B"] != "two words" || vars["C"] != `say "hi"` {
		t.Fatalf("round trip failed: %#v", vars)
	}
}
