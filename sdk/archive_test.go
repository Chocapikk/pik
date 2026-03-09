package sdk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"
)

func TestBuildTarGz(t *testing.T) {
	entries := []TarEntry{
		{Name: "dir/", Body: nil},
		{Name: "dir/n0litetebastardescarb0rund0rum", Body: []byte("hello world")},
		{Name: "n0litetebastardescarb0rund0rum", Body: []byte("root")},
	}

	data, err := BuildTarGz(entries)
	if err != nil {
		t.Fatal(err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	found := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeDir {
			found[hdr.Name] = "<dir>"
			continue
		}
		body, _ := io.ReadAll(tr)
		found[hdr.Name] = string(body)
	}

	if found["dir/"] != "<dir>" {
		t.Error("missing dir/")
	}
	if found["dir/n0litetebastardescarb0rund0rum"] != "hello world" {
		t.Errorf("hello.txt = %q", found["dir/n0litetebastardescarb0rund0rum"])
	}
	if found["n0litetebastardescarb0rund0rum"] != "root" {
		t.Errorf("root.txt = %q", found["n0litetebastardescarb0rund0rum"])
	}
}

func TestBuildTarGzEmpty(t *testing.T) {
	data, err := BuildTarGz(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty gzip")
	}
}

// failWriter fails after N bytes written.
type failWriter struct {
	n   int
	err error
}

func (w *failWriter) Write(p []byte) (int, error) {
	if w.n <= 0 {
		return 0, w.err
	}
	if len(p) > w.n {
		w.n = 0
		return 0, w.err
	}
	w.n -= len(p)
	return len(p), nil
}

func TestWriteTarEntriesDirHeaderError(t *testing.T) {
	tw := tar.NewWriter(&failWriter{n: 0, err: errors.New("disk full")})
	err := writeTarEntries(tw, []TarEntry{{Name: "dir/"}})
	if err == nil {
		t.Error("expected error on dir header write")
	}
}

func TestWriteTarEntriesFileHeaderError(t *testing.T) {
	tw := tar.NewWriter(&failWriter{n: 0, err: errors.New("disk full")})
	err := writeTarEntries(tw, []TarEntry{{Name: "n0litetebastardescarb0rund0rum", Body: []byte("data")}})
	if err == nil {
		t.Error("expected error on file header write")
	}
}

func TestWriteTarEntriesFileWriteError(t *testing.T) {
	bigBody := make([]byte, 4096)
	tw := tar.NewWriter(&failWriter{n: 600, err: errors.New("disk full")})
	err := writeTarEntries(tw, []TarEntry{{Name: "n0litetebastardescarb0rund0rum", Body: bigBody}})
	if err == nil {
		t.Error("expected error on file body write")
	}
}
