package sdk

import (
	"context"
	"time"
)

// LabManager is implemented by pkg/lab and registered via SetLabManager.
// This late binding keeps Docker SDK out of the sdk package and out of
// standalone binaries that don't import pkg/lab.
type LabManager interface {
	Start(ctx context.Context, name string, services []Service) error
	Stop(ctx context.Context, name string) error
	Status(ctx context.Context) ([]LabStatus, error)
	IsRunning(ctx context.Context, name string) bool
	Target(ctx context.Context, name string) string
	WaitReady(ctx context.Context, addr string, timeout time.Duration) error
	WaitProbe(ctx context.Context, timeout time.Duration, fn func() error) error
	DockerGateway() string
}

// LabStatus holds status for a lab (mirrors lab.LabInfo without importing it).
type LabStatus struct {
	Name     string
	Services []LabServiceStatus
}

// LabServiceStatus holds status for one service container.
type LabServiceStatus struct {
	Name  string
	Image string
	State string
	Ports string
}

var labMgr LabManager

// SetLabManager registers the lab manager (called from pkg/lab init).
func SetLabManager(m LabManager) { labMgr = m }

// GetLabManager returns the registered lab manager, or nil if not available.
func GetLabManager() LabManager { return labMgr }
