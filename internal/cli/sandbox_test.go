package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout runs fn while redirecting os.Stdout to a buffer and returns
// the captured output. Used to assert on subcommand printed output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	prev := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = prev }()

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()
	require.NoError(t, w.Close())
	<-done
	return buf.String()
}

func TestSandboxName_DerivesNameFromTicketID(t *testing.T) {
	app := NewApp(&version.BuildInfo{})
	out := captureStdout(t, func() {
		err := app.RunContext(context.Background(), []string{"brains", "sandbox", "name", "DEV-123"})
		require.NoError(t, err)
	})
	assert.Equal(t, "zk-dev-123\n", out)
}

func TestSandboxName_MissingTicketID_Errors(t *testing.T) {
	app := NewApp(&version.BuildInfo{})
	err := app.RunContext(context.Background(), []string{"brains", "sandbox", "name"})
	require.Error(t, err)
}

func TestSandboxCleanup_IsIdempotent(t *testing.T) {
	// sandbox.Cleanup is documented idempotent: silent success when sbx is
	// missing or the named sandbox does not exist.
	app := NewApp(&version.BuildInfo{})
	err := app.RunContext(context.Background(), []string{"brains", "sandbox", "cleanup", "DEV-NONEXISTENT-9999"})
	require.NoError(t, err)
}

func TestFilterZKPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"only matches", "zk-a\nzk-b\n", []string{"zk-a", "zk-b"}},
		{"mixed", "zk-foo\nother\nzk-bar\nstuff\n", []string{"zk-foo", "zk-bar"}},
		{"none match", "alpha\nbeta\n", nil},
		{"trims whitespace", "  zk-spaced  \n", []string{"zk-spaced"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, filterZKPrefix(tc.in))
		})
	}
}
