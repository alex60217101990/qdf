package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
)

// embURL and embSHA256 are intentionally empty. Pin a public .f32 embedding
// corpus URL and its sha256 here when a real download is desired; the harness
// falls back to synthetic Gaussian vectors when these are unset.
const (
	embURL    = "" // TODO-at-exec: pin a public .f32 embedding URL
	embSHA256 = "" // TODO-at-exec: pin its sha256
)

// loadSynthetic generates n unit-Gaussian vectors of the given dimension.
func loadSynthetic(n, dim int, seed int64) [][]float64 {
	r := rand.New(rand.NewSource(seed))
	out := make([][]float64, n)
	for i := range out {
		v := make([]float64, dim)
		for j := range v {
			v[j] = r.NormFloat64()
		}
		out[i] = v
	}
	return out
}

// fetchPinned downloads the configured corpus URL to path and verifies the
// sha256 checksum. Returns an error if embURL is empty or the checksum does
// not match.
func fetchPinned(path string) error {
	if embURL == "" {
		return errors.New("no pinned corpus configured")
	}
	resp, err := http.Get(embURL) //nolint:gosec // URL is a compile-time constant
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != embSHA256 {
		return errors.New("corpus sha256 mismatch")
	}
	return os.WriteFile(path, body, 0o644)
}
