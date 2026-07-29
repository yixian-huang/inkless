package themecatalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultVerifyTimeout = 12 * time.Second
const defaultVerifyMaxBytes = 8 << 20 // 8 MiB UMD cap for hash verify

// VerifyUMDSHA256 downloads umdURL (HTTPS allowlisted) and checks sha256 hex digest.
// Empty expectedSHA skips verification (returns nil).
func VerifyUMDSHA256(ctx context.Context, umdURL, expectedSHA string, allowHosts []string) error {
	expectedSHA = strings.TrimSpace(strings.ToLower(expectedSHA))
	if expectedSHA == "" {
		return nil
	}
	if err := ValidateUMDURL(umdURL, allowHosts); err != nil {
		return err
	}

	client := &http.Client{Timeout: defaultVerifyTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, umdURL, nil)
	if err != nil {
		return fmt.Errorf("build umd request: %w", err)
	}
	req.Header.Set("User-Agent", "inkless-theme-catalog/1")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download umd for hash: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download umd for hash: status %d", resp.StatusCode)
	}

	h := sha256.New()
	limited := io.LimitReader(resp.Body, defaultVerifyMaxBytes+1)
	n, err := io.Copy(h, limited)
	if err != nil {
		return fmt.Errorf("read umd body: %w", err)
	}
	if n > defaultVerifyMaxBytes {
		return fmt.Errorf("umd body exceeds %d bytes", defaultVerifyMaxBytes)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedSHA {
		return fmt.Errorf("umd sha256 mismatch: expected %s, got %s", expectedSHA, got)
	}
	return nil
}
