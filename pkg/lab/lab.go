package lab

import (
	"context"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/sdk"
)

const (
	labelLab     = "pik.lab"
	labelService = "pik.lab.service"
)

// --- Docker client ---

func withClient(fn func(cli *client.Client) error) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer cli.Close()
	return fn(cli)
}

// --- Poll loop ---

// poll retries fn every interval until it returns nil or the deadline passes.
func poll(ctx context.Context, timeout, interval time.Duration, fn func() error) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		time.Sleep(interval)
	}
	return lastErr
}

// --- Public API ---

// Start creates and starts containers for the given lab services.
//
// Each service gets a network alias matching its Name so services
// can reference each other by short name (like docker compose).
// Services with a healthcheck are waited on before starting the
// next service, giving dependencies time to become ready.
func Start(ctx context.Context, name string, services []sdk.Service) error {
	if len(services) == 0 {
		return fmt.Errorf("lab has no services")
	}

	return withClient(func(cli *client.Client) error {
		teardown(ctx, cli, name)

		netName := "pik_" + name
		netResp, err := cli.NetworkCreate(ctx, netName, network.CreateOptions{
			Labels: map[string]string{labelLab: name},
		})
		if err != nil {
			return fmt.Errorf("create network: %w", err)
		}

		for _, svc := range services {
			if svc.Config.Image == "" {
				return fmt.Errorf("service %q: no image specified", svc.Name)
			}

			output.Status("Pulling %s", svc.Config.Image)
			reader, err := cli.ImagePull(ctx, svc.Config.Image, image.PullOptions{})
			if err != nil {
				return fmt.Errorf("pull %s: %w", svc.Config.Image, err)
			}
			io.Copy(io.Discard, reader)
			reader.Close()

			cfg := svc.Config
			if cfg.Labels == nil {
				cfg.Labels = make(map[string]string)
			}
			cfg.Labels[labelLab] = name
			cfg.Labels[labelService] = svc.Name

			netCfg := &network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					netName: {
						NetworkID: netResp.ID,
						Aliases:   []string{svc.Name},
					},
				},
			}

			containerName := "pik-" + name + "-" + svc.Name
			output.Status("Creating %s", containerName)
			resp, err := cli.ContainerCreate(ctx, &cfg, &svc.HostConfig, netCfg, nil, containerName)
			if err != nil {
				return fmt.Errorf("create %s: %w", containerName, err)
			}

			if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("start %s: %w", containerName, err)
			}
			output.Success("Started %s (%s)", containerName, svc.Config.Image)

			if svc.Config.Healthcheck != nil {
				output.Status("Waiting for %s to be healthy", svc.Name)
				err := poll(ctx, 120*time.Second, 2*time.Second, func() error {
					inspect, err := cli.ContainerInspect(ctx, resp.ID)
					if err != nil {
						return err
					}
					if inspect.State.Health == nil {
						return fmt.Errorf("no health status yet")
					}
					switch inspect.State.Health.Status {
					case "healthy":
						return nil
					case "unhealthy":
						return fmt.Errorf("container unhealthy")
					}
					return fmt.Errorf("health: %s", inspect.State.Health.Status)
				})
				if err != nil {
					output.Warning("%s: %v", svc.Name, err)
				}
			}
		}
		return nil
	})
}

// Stop tears down all containers and the network for a lab.
func Stop(ctx context.Context, name string) error {
	return withClient(func(cli *client.Client) error {
		teardown(ctx, cli, name)
		return nil
	})
}

// --- Status ---

// LabInfo holds status for a lab.
type LabInfo struct {
	Name     string
	Services []ServiceInfo
}

// ServiceInfo holds status for one service container.
type ServiceInfo struct {
	Name  string
	Image string
	State string
	Ports string
}

// Status returns info for all labs tracked by pik.
func Status(ctx context.Context) ([]LabInfo, error) {
	var labs []LabInfo
	err := withClient(func(cli *client.Client) error {
		containers, err := cli.ContainerList(ctx, container.ListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.Arg("label", labelLab)),
		})
		if err != nil {
			return err
		}
		if len(containers) == 0 {
			return nil
		}

		byLab := make(map[string][]ServiceInfo)
		for _, ctr := range containers {
			byLab[ctr.Labels[labelLab]] = append(byLab[ctr.Labels[labelLab]], ServiceInfo{
				Name:  ctr.Labels[labelService],
				Image: ctr.Image,
				State: ctr.State,
				Ports: formatPorts(ctr.Ports),
			})
		}
		for labName, services := range byLab {
			labs = append(labs, LabInfo{Name: labName, Services: services})
		}
		sort.Slice(labs, func(i, j int) bool { return labs[i].Name < labs[j].Name })
		return nil
	})
	return labs, err
}

// IsRunning checks if a lab has running containers.
func IsRunning(ctx context.Context, name string) bool {
	running := false
	withClient(func(cli *client.Client) error {
		containers, err := cli.ContainerList(ctx, container.ListOptions{
			Filters: filters.NewArgs(
				filters.Arg("label", labelLab+"="+name),
				filters.Arg("status", "running"),
			),
		})
		running = err == nil && len(containers) > 0
		return nil
	})
	return running
}

// --- Readiness ---

// Target derives the exploit target (127.0.0.1:port) from a lab's
// service definitions by finding the first host port binding.
func Target(services []sdk.Service) string {
	for _, svc := range services {
		for _, bindings := range svc.HostConfig.PortBindings {
			for _, binding := range bindings {
				if binding.HostPort != "" {
					return "127.0.0.1:" + binding.HostPort
				}
			}
		}
	}
	return "127.0.0.1"
}

// WaitReady polls a TCP address until it accepts connections.
func WaitReady(ctx context.Context, addr string, timeout time.Duration) error {
	return poll(ctx, timeout, 2*time.Second, func() error {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	})
}

// WaitProbe retries fn until it returns nil or timeout expires.
// Use with the module's Check() as fn to wait for the application
// layer to be fully ready (not just TCP).
func WaitProbe(ctx context.Context, timeout time.Duration, fn func() error) error {
	return poll(ctx, timeout, 3*time.Second, fn)
}

// DockerGateway returns the default Docker bridge gateway IP.
func DockerGateway() string {
	conn, err := net.DialTimeout("tcp", "172.17.0.1:1", 100*time.Millisecond)
	if conn != nil {
		conn.Close()
	}
	if err != nil && strings.Contains(err.Error(), "no route") {
		return ""
	}
	return "172.17.0.1"
}

// --- Internal ---

func teardown(ctx context.Context, cli *client.Client, name string) {
	containers, _ := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", labelLab+"="+name)),
	})
	for _, ctr := range containers {
		cli.ContainerStop(ctx, ctr.ID, container.StopOptions{})
		cli.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{Force: true})
	}

	networks, _ := cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", labelLab+"="+name)),
	})
	for _, n := range networks {
		cli.NetworkRemove(ctx, n.ID)
	}
}

func formatPorts(ports []container.Port) string {
	var parts []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			parts = append(parts, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
}
