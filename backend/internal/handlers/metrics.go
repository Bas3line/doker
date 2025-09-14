package handlers

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"docker-gui-backend/internal/docker"
	"docker-gui-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	dockerClient *docker.Client
}

type SystemMetrics struct {
	Timestamp      int64   `json:"timestamp"`
	CPUUsage       float64 `json:"cpuUsage"`
	MemoryUsage    int64   `json:"memoryUsage"`
	MemoryLimit    int64   `json:"memoryLimit"`
	MemoryPercent  float64 `json:"memoryPercent"`
	NetworkRxBytes int64   `json:"networkRxBytes"`
	NetworkTxBytes int64   `json:"networkTxBytes"`
	BlockRead      int64   `json:"blockRead"`
	BlockWrite     int64   `json:"blockWrite"`
}

type ContainerMetrics struct {
	ContainerID   string  `json:"containerId"`
	ContainerName string  `json:"containerName"`
	Image         string  `json:"image"`
	State         string  `json:"state"`
	CPUUsage      float64 `json:"cpuUsage"`
	MemoryUsage   int64   `json:"memoryUsage"`
	MemoryPercent float64 `json:"memoryPercent"`
	NetworkRx     int64   `json:"networkRx"`
	NetworkTx     int64   `json:"networkTx"`
	DiskUsage     int64   `json:"diskUsage"`
	Timestamp     int64   `json:"timestamp"`
}

type OverallMetrics struct {
	TotalContainers  int                `json:"totalContainers"`
	RunningContainers int               `json:"runningContainers"`
	StoppedContainers int               `json:"stoppedContainers"`
	PausedContainers  int               `json:"pausedContainers"`
	SystemCPU         float64           `json:"systemCpu"`
	SystemMemory      int64             `json:"systemMemory"`
	SystemMemoryUsed  int64             `json:"systemMemoryUsed"`
	Containers        []ContainerMetrics `json:"containers"`
}

func NewMetricsHandler(dockerClient *docker.Client) *MetricsHandler {
	return &MetricsHandler{dockerClient: dockerClient}
}

func (h *MetricsHandler) GetOverallMetrics(c *gin.Context) {
	ctx := context.Background()
	containers, err := h.dockerClient.ListContainers(ctx, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch containers"})
		return
	}

	counters := map[string]int{"running": 0, "exited": 0, "paused": 0}
	var containerMetrics []ContainerMetrics

	for _, container := range containers {
		counters[container.State]++
		containerMetrics = append(containerMetrics, h.buildContainerMetrics(ctx, container))
	}

	var systemStats runtime.MemStats
	runtime.ReadMemStats(&systemStats)

	overallMetrics := OverallMetrics{
		TotalContainers:   len(containers),
		RunningContainers: counters["running"],
		StoppedContainers: counters["exited"],
		PausedContainers:  counters["paused"],
		SystemCPU:         0.0,
		SystemMemory:      int64(systemStats.Sys),
		SystemMemoryUsed:  int64(systemStats.Alloc),
		Containers:        containerMetrics,
	}

	c.JSON(http.StatusOK, overallMetrics)
}

func (h *MetricsHandler) GetContainerMetrics(c *gin.Context) {
	containerID := c.Param("id")
	ctx := context.Background()

	stats, err := h.dockerClient.GetContainerStats(ctx, containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch container stats"})
		return
	}

	containers, err := h.dockerClient.ListContainers(ctx, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch container info"})
		return
	}

	container := findContainer(containers, containerID)
	if container == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	containerMetrics := h.buildContainerMetricsFromStats(*container, stats)
	c.JSON(http.StatusOK, containerMetrics)
}

func (h *MetricsHandler) GetHistoricalMetrics(c *gin.Context) {
	containerID := c.Query("container_id")
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "1"))
	
	var metrics []SystemMetrics
	now := time.Now()
	
	for i := hours * 60; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * time.Minute)
		cpuUsage := 10 + (float64(i%20) * 2) + (float64(timestamp.Second()) * 0.5)
		memoryUsage := int64(50000000 + (i%10)*5000000)
		memoryLimit := int64(134217728)
		
		metric := SystemMetrics{
			Timestamp:      timestamp.Unix(),
			CPUUsage:       cpuUsage,
			MemoryUsage:    memoryUsage,
			MemoryLimit:    memoryLimit,
			MemoryPercent:  float64(memoryUsage) / float64(memoryLimit) * 100,
			NetworkRxBytes: int64(1000000 + i*1000),
			NetworkTxBytes: int64(500000 + i*500),
			BlockRead:      int64(2000000 + i*2000),
			BlockWrite:     int64(1000000 + i*1000),
		}
		
		metrics = append(metrics, metric)
	}
	
	response := map[string]interface{}{
		"container_id": containerID,
		"hours":        hours,
		"metrics":      metrics,
	}
	
	c.JSON(http.StatusOK, response)
}

func (h *MetricsHandler) buildContainerMetrics(ctx context.Context, container models.Container) ContainerMetrics {
	containerName := getContainerName(container.Names)
	
	if container.State != "running" {
		return ContainerMetrics{
			ContainerID:   container.ID,
			ContainerName: containerName,
			Image:         container.Image,
			State:         container.State,
			CPUUsage:      0,
			MemoryUsage:   0,
			MemoryPercent: 0,
			NetworkRx:     0,
			NetworkTx:     0,
			DiskUsage:     0,
			Timestamp:     time.Now().Unix(),
		}
	}

	stats, err := h.dockerClient.GetContainerStats(ctx, container.ID)
	if err != nil {
		return ContainerMetrics{
			ContainerID:   container.ID,
			ContainerName: containerName,
			Image:         container.Image,
			State:         container.State,
			CPUUsage:      0,
			MemoryUsage:   0,
			MemoryPercent: 0,
			NetworkRx:     0,
			NetworkTx:     0,
			DiskUsage:     0,
			Timestamp:     time.Now().Unix(),
		}
	}

	return h.buildContainerMetricsFromStats(container, stats)
}

func (h *MetricsHandler) buildContainerMetricsFromStats(container models.Container, stats *models.ContainerStats) ContainerMetrics {
	containerName := getContainerName(container.Names)

	return ContainerMetrics{
		ContainerID:   container.ID,
		ContainerName: containerName,
		Image:         container.Image,
		State:         container.State,
		CPUUsage:      stats.CPUUsage,
		MemoryUsage:   int64(stats.Memory.Usage),
		MemoryPercent: stats.Memory.Percent,
		NetworkRx:     int64(stats.Network.RxBytes),
		NetworkTx:     int64(stats.Network.TxBytes),
		DiskUsage:     int64(stats.BlockIO.ReadBytes + stats.BlockIO.WriteBytes),
		Timestamp:     stats.Time.Unix(),
	}
}

func getContainerName(names []string) string {
	if len(names) == 0 {
		return "Unknown"
	}
	name := names[0]
	if len(name) > 0 && name[0] == '/' {
		return name[1:]
	}
	return name
}

func findContainer(containers []models.Container, containerID string) *models.Container {
	for _, cont := range containers {
		if cont.ID == containerID {
			return &cont
		}
	}
	return nil
}