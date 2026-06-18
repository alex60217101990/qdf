package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sampleDataTarball is the GitHub codeload endpoint for a gzip tarball of the
// adalanche-sampledata repository's default branch. One request fetches the
// whole repo; we extract only the localmachine JSON dumps and discard the rest.
const sampleDataTarball = "https://codeload.github.com/lkarlslund/adalanche-sampledata/tar.gz/refs/heads/main"

// localmachineSuffix is the in-tarball path fragment that selects the dumps the
// benchmark uses; everything else in the archive (AD msgp.lz4 dumps, docs) is
// skipped.
const localmachineSuffix = "/goad/localmachine/"

// fetchSampleData downloads the adalanche-sampledata tarball into a fresh temp
// directory, extracts only the goad/localmachine/*.json dumps into it, and
// returns that directory plus a cleanup func that removes the whole temp tree.
// The tarball stream is consumed in memory and never written to disk, so no
// archive lingers; cleanup drops the extracted JSONs once the run is done.
func fetchSampleData() (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "qdf-bench-sampledata-")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(sampleDataTarball)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download sample data: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		cleanup()
		return "", nil, fmt.Errorf("download sample data: %s", resp.Status)
	}

	n, err := extractLocalmachineJSON(resp.Body, dir)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if n == 0 {
		cleanup()
		return "", nil, fmt.Errorf("no localmachine JSON files found in %s", sampleDataTarball)
	}
	return dir, cleanup, nil
}

// extractLocalmachineJSON streams a gzip tarball from r and writes every
// goad/localmachine/*.json entry into dir (flattened to its base name),
// returning the count extracted. Non-matching entries are skipped without being
// buffered.
func extractLocalmachineJSON(r io.Reader, dir string) (int, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return 0, fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	count := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if !strings.Contains(hdr.Name, localmachineSuffix) || !strings.HasSuffix(hdr.Name, ".json") {
			continue
		}
		// Flatten to the base name; the loader globs dir/*.json.
		dst := filepath.Join(dir, filepath.Base(hdr.Name))
		f, err := os.Create(dst)
		if err != nil {
			return count, fmt.Errorf("create %s: %w", dst, err)
		}
		// Bound the copy by the declared header size so a hostile archive
		// cannot stream an unbounded body into the temp file.
		if _, err := io.CopyN(f, tr, hdr.Size); err != nil && err != io.EOF {
			f.Close()
			return count, fmt.Errorf("extract %s: %w", dst, err)
		}
		if err := f.Close(); err != nil {
			return count, fmt.Errorf("close %s: %w", dst, err)
		}
		count++
	}
	return count, nil
}
