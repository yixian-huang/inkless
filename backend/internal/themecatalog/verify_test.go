package themecatalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyUMDSHA256_SkipEmpty(t *testing.T) {
	require.NoError(t, VerifyUMDSHA256(context.Background(), "https://github.com/x/y.js", "", DefaultUMDAllowHosts))
}

func TestVerifyUMDSHA256_RejectsBadHost(t *testing.T) {
	require.Error(t, VerifyUMDSHA256(context.Background(), "https://evil.example/t.js", "abc", DefaultUMDAllowHosts))
}
