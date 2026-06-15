package system

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	out string
	err error
}

func (f fakeRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return []byte(f.out), f.err
}

func TestPort443Checks(t *testing.T) {
	ctx := context.Background()
	if err := Port443Conflict(ctx, fakeRunner{out: `LISTEN 0 4096 *:443 users:(("nginx",pid=1))`}); err == nil {
		t.Fatal("expected conflict")
	}
	if err := Port443Conflict(ctx, fakeRunner{out: `LISTEN 0 4096 *:443 users:(("xray",pid=1))`}); err != nil {
		t.Fatal(err)
	}
	if err := Port443OwnedByXray(ctx, fakeRunner{out: `LISTEN 0 4096 *:443 users:(("xray",pid=1))`}); err != nil {
		t.Fatal(err)
	}
	if err := Port443OwnedByXray(ctx, fakeRunner{err: errors.New("failed")}); err == nil || !strings.Contains(err.Error(), "inspect") {
		t.Fatal("expected inspection error")
	}
}
