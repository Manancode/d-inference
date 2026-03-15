package posture

import (
	"os/exec"
	"strings"
	"time"

	"github.com/dginf/dginf/services/providerd/internal/domain"
)

type Runner interface {
	Run(name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type Collector struct {
	runner Runner
	now    func() time.Time
}

func NewCollector(now func() time.Time) *Collector {
	if now == nil {
		now = time.Now
	}
	return &Collector{
		runner: execRunner{},
		now:    now,
	}
}

func (c *Collector) Snapshot() domain.PostureReport {
	return domain.PostureReport{
		OSVersion:       c.commandText("sw_vers", "-productVersion"),
		SIPStatus:       normalizeStatus(c.commandText("csrutil", "status")),
		FileVaultStatus: normalizeStatus(c.commandText("fdesetup", "status")),
		CollectedAt:     c.now().UTC(),
	}
}

func (c *Collector) commandText(name string, args ...string) string {
	output, err := c.runner.Run(name, args...)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func normalizeStatus(raw string) string {
	if raw == "" {
		return "unknown"
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "enabled"):
		return "enabled"
	case strings.Contains(lower, "disabled"):
		return "disabled"
	case strings.Contains(lower, "on"):
		return "enabled"
	case strings.Contains(lower, "off"):
		return "disabled"
	default:
		return raw
	}
}
