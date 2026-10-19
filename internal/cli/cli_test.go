package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestVerifyChecksum(t *testing.T) {
	data := []byte("payload")
	sum := sha256.Sum256(data)
	sums := []byte(hex.EncodeToString(sum[:]) + "  ppm_linux_amd64.tar.gz\n")
	if err := verifyChecksum(data, sums, "ppm_linux_amd64.tar.gz"); err != nil {
		t.Fatal(err)
	}
	if verifyChecksum([]byte("tampered"), sums, "ppm_linux_amd64.tar.gz") == nil {
		t.Fatal("tampered archive accepted")
	}
	if verifyChecksum(data, sums, "other.zip") == nil {
		t.Fatal("unlisted archive accepted")
	}
}

func TestExtractBinaryTarGz(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{"README.md": "readme", "ppm": "binary"} {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	bin, err := extractBinary(buf.Bytes(), "tar.gz")
	if err != nil || string(bin) != "binary" {
		t.Fatalf("extractBinary = %q, %v", bin, err)
	}
}

func TestMatchesClean(t *testing.T) {
	fsys := fstest.MapFS{
		"__pycache__/x.pyc": {}, "build/a": {}, "src/build/b": {}, "pkg.egg-info/c": {}, "main.py": {},
	}
	entry := func(path string) fs.DirEntry {
		info, err := fs.Stat(fsys, path)
		if err != nil {
			t.Fatal(err)
		}
		return fs.FileInfoToDirEntry(info)
	}
	cases := []struct {
		path   string
		atRoot bool
		want   bool
	}{
		{"__pycache__", false, true},
		{"build", true, true},
		{"src/build", false, false}, // a package may be named build
		{"pkg.egg-info", true, true},
		{"main.py", true, false},
	}
	for _, c := range cases {
		if got := matchesClean(entry(c.path), c.atRoot); got != c.want {
			t.Errorf("matchesClean(%s, root=%v) = %v, want %v", c.path, c.atRoot, got, c.want)
		}
	}
}
