package posture

import (
	"testing"
	"time"
)

type fakeRunner struct {
	outputs map[string][]byte
}

func (f fakeRunner) Run(name string, args ...string) ([]byte, error) {
	return f.outputs[name], nil
}

func TestSnapshotNormalizesStatuses(t *testing.T) {
	collector := &Collector{
		runner: fakeRunner{
			outputs: map[string][]byte{
				"sw_vers":  []byte("14.7.1\n"),
				"csrutil":  []byte("System Integrity Protection status: enabled.\n"),
				"fdesetup": []byte("FileVault is Off.\n"),
			},
		},
		now: func() time.Time { return time.Unix(1_700_000_000, 0) },
	}
	snapshot := collector.Snapshot()
	if snapshot.OSVersion != "14.7.1" {
		t.Fatalf("unexpected OS version: %q", snapshot.OSVersion)
	}
	if snapshot.SIPStatus != "enabled" {
		t.Fatalf("unexpected SIP status: %q", snapshot.SIPStatus)
	}
	if snapshot.FileVaultStatus != "disabled" {
		t.Fatalf("unexpected FileVault status: %q", snapshot.FileVaultStatus)
	}
}
