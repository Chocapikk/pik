package sdk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
)

// TarEntry is a file or directory entry for BuildTarGz.
type TarEntry struct {
	Name string // path inside archive (directories end with /)
	Body []byte // nil for directories
}

// BuildTarGz creates a tar.gz archive from the given entries.
func BuildTarGz(entries []TarEntry) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		if len(e.Body) == 0 && (len(e.Name) == 0 || e.Name[len(e.Name)-1] == '/') {
			if err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     e.Name,
				Mode:     0755,
			}); err != nil {
				return nil, err
			}
			continue
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: e.Name,
			Mode: 0644,
			Size: int64(len(e.Body)),
		}); err != nil {
			return nil, err
		}
		if _, err := tw.Write(e.Body); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
