package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"docker-gui-backend/pkg/models"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

func (c *Client) ListContainers(ctx context.Context, all bool) ([]models.Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, err
	}

	var result []models.Container
	for _, container := range containers {
		ports := make([]models.Port, len(container.Ports))
		for i, port := range container.Ports {
			ports[i] = models.Port{
				IP:          port.IP,
				PrivatePort: port.PrivatePort,
				PublicPort:  port.PublicPort,
				Type:        port.Type,
			}
		}

		mounts := make([]models.Mount, len(container.Mounts))
		for i, mount := range container.Mounts {
			mounts[i] = models.Mount{
				Type:        string(mount.Type),
				Name:        mount.Name,
				Source:      mount.Source,
				Destination: mount.Destination,
				Driver:      mount.Driver,
				Mode:        mount.Mode,
				RW:          mount.RW,
				Propagation: string(mount.Propagation),
			}
		}

		result = append(result, models.Container{
			ID:      container.ID,
			Names:   container.Names,
			Image:   container.Image,
			ImageID: container.ImageID,
			Command: container.Command,
			Created: container.Created,
			Ports:   ports,
			Labels:  container.Labels,
			State:   container.State,
			Status:  container.Status,
			Mounts:  mounts,
		})
	}

	return result, nil
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	return c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RestartContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return c.cli.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: force})
}

func (c *Client) GetContainerLogs(ctx context.Context, containerID string, lines int) ([]models.LogEntry, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       fmt.Sprintf("%d", lines),
	}

	logs, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, err
	}
	defer logs.Close()

	content, err := io.ReadAll(logs)
	if err != nil {
		return nil, err
	}

	var logEntries []models.LogEntry
	lines_str := strings.Split(string(content), "\n")
	
	for _, line := range lines_str {
		if len(line) == 0 {
			continue
		}
		
		var stream string
		var message string
		
		if len(line) >= 8 {
			switch line[0] {
			case 1:
				stream = "stdout"
			case 2:
				stream = "stderr"
			default:
				stream = "stdout"
			}
			message = line[8:]
		} else {
			stream = "stdout"
			message = line
		}
		
		var timestamp time.Time
		if strings.Contains(message, "T") && (strings.Contains(message, "Z") || strings.Contains(message, "+")) {
			parts := strings.SplitN(message, " ", 2)
			if len(parts) == 2 {
				if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					timestamp = parsedTime
					message = parts[1]
				} else if parsedTime, err := time.Parse(time.RFC3339, parts[0]); err == nil {
					timestamp = parsedTime
					message = parts[1]
				} else {
					timestamp = time.Now()
				}
			} else {
				timestamp = time.Now()
			}
		} else {
			timestamp = time.Now()
		}
		
		if strings.TrimSpace(message) != "" {
			logEntries = append(logEntries, models.LogEntry{
				Timestamp: timestamp,
				Message:   strings.TrimSpace(message),
				Stream:    stream,
			})
		}
	}

	return logEntries, nil
}

func (c *Client) GetContainerStats(ctx context.Context, containerID string) (*models.ContainerStats, error) {
	stats, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var dockerStats types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&dockerStats); err != nil {
		return nil, err
	}

	cpuUsage := calculateCPUPercent(&dockerStats)
	memoryUsage := dockerStats.MemoryStats.Usage
	memoryLimit := dockerStats.MemoryStats.Limit
	memoryPercent := float64(memoryUsage) / float64(memoryLimit) * 100

	var rxBytes, txBytes, rxPackets, txPackets uint64
	for _, network := range dockerStats.Networks {
		rxBytes += network.RxBytes
		txBytes += network.TxBytes
		rxPackets += network.RxPackets
		txPackets += network.TxPackets
	}

	var readBytes, writeBytes uint64
	for _, blkio := range dockerStats.BlkioStats.IoServiceBytesRecursive {
		if blkio.Op == "Read" {
			readBytes += blkio.Value
		} else if blkio.Op == "Write" {
			writeBytes += blkio.Value
		}
	}

	return &models.ContainerStats{
		ID:       containerID,
		CPUUsage: cpuUsage,
		Memory: models.Memory{
			Usage:   memoryUsage,
			Limit:   memoryLimit,
			Percent: memoryPercent,
		},
		Network: models.Network{
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			RxPackets: rxPackets,
			TxPackets: txPackets,
		},
		BlockIO: models.BlockIO{
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
		},
		Time: time.Now(),
	}, nil
}

func calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	
	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100
	}
	return 0
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) ListImages(ctx context.Context) ([]models.Image, error) {
	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []models.Image
	for _, img := range images {
		var repoTags []string
		if len(img.RepoTags) > 0 {
			repoTags = img.RepoTags
		} else {
			repoTags = []string{"<none>:<none>"}
		}

		result = append(result, models.Image{
			ID:       img.ID,
			RepoTags: repoTags,
			Size:     img.Size,
			Created:  img.Created,
		})
	}

	return result, nil
}

func (c *Client) PullImage(ctx context.Context, imageName string) error {
	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	
	_, err = io.ReadAll(reader)
	return err
}

func (c *Client) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := c.cli.ImageRemove(ctx, imageID, image.RemoveOptions{Force: force})
	return err
}

func (c *Client) PruneImages(ctx context.Context) error {
	_, err := c.cli.ImagesPrune(ctx, filters.NewArgs())
	return err
}