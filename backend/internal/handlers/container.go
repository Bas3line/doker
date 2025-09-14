package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"docker-gui-backend/internal/database"
	"docker-gui-backend/internal/docker"
	"docker-gui-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

func (h *ContainerHandler) GetActivityLogs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}
	
	logs, err := h.db.GetAllLogs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *ContainerHandler) GetContainerActivityLogs(c *gin.Context) {
	containerID := c.Param("id")
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}
	
	logs, err := h.db.GetContainerLogs(containerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

type ContainerHandler struct {
	dockerClient *docker.Client
	db           *database.DB
}

func NewContainerHandler(dockerClient *docker.Client, db *database.DB) *ContainerHandler {
	return &ContainerHandler{
		dockerClient: dockerClient,
		db:           db,
	}
}

func (h *ContainerHandler) ListContainers(c *gin.Context) {
	all := c.DefaultQuery("all", "true") == "true"
	
	containers, err := h.dockerClient.ListContainers(c.Request.Context(), all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, containers)
}

func (h *ContainerHandler) StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	containers, _ := h.dockerClient.ListContainers(c.Request.Context(), true)
	containerName := "unknown"
	for _, container := range containers {
		if container.ID == containerID {
			if len(container.Names) > 0 {
				containerName = container.Names[0]
			}
			break
		}
	}
	
	err := h.dockerClient.StartContainer(c.Request.Context(), containerID)
	if err != nil {
		h.db.LogContainerAction(containerID, containerName, "start_failed", "docker-gui", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.LogContainerAction(containerID, containerName, "start", "docker-gui", "Container started successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Container started successfully"})
}

func (h *ContainerHandler) StopContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	containers, _ := h.dockerClient.ListContainers(c.Request.Context(), true)
	containerName := "unknown"
	for _, container := range containers {
		if container.ID == containerID {
			if len(container.Names) > 0 {
				containerName = container.Names[0]
			}
			break
		}
	}
	
	err := h.dockerClient.StopContainer(c.Request.Context(), containerID)
	if err != nil {
		h.db.LogContainerAction(containerID, containerName, "stop_failed", "docker-gui", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.LogContainerAction(containerID, containerName, "stop", "docker-gui", "Container stopped successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Container stopped successfully"})
}

func (h *ContainerHandler) RestartContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	containers, _ := h.dockerClient.ListContainers(c.Request.Context(), true)
	containerName := "unknown"
	for _, container := range containers {
		if container.ID == containerID {
			if len(container.Names) > 0 {
				containerName = container.Names[0]
			}
			break
		}
	}
	
	err := h.dockerClient.RestartContainer(c.Request.Context(), containerID)
	if err != nil {
		h.db.LogContainerAction(containerID, containerName, "restart_failed", "docker-gui", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.LogContainerAction(containerID, containerName, "restart", "docker-gui", "Container restarted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Container restarted successfully"})
}

func (h *ContainerHandler) RemoveContainer(c *gin.Context) {
	containerID := c.Param("id")
	force := c.DefaultQuery("force", "false") == "true"
	
	containers, _ := h.dockerClient.ListContainers(c.Request.Context(), true)
	containerName := "unknown"
	for _, container := range containers {
		if container.ID == containerID {
			if len(container.Names) > 0 {
				containerName = container.Names[0]
			}
			break
		}
	}
	
	err := h.dockerClient.RemoveContainer(c.Request.Context(), containerID, force)
	if err != nil {
		h.db.LogContainerAction(containerID, containerName, "remove_failed", "docker-gui", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	details := fmt.Sprintf("Container removed successfully (force: %v)", force)
	h.db.LogContainerAction(containerID, containerName, "remove", "docker-gui", details)
	c.JSON(http.StatusOK, gin.H{"message": "Container removed successfully"})
}

func (h *ContainerHandler) GetContainerLogs(c *gin.Context) {
	containerID := c.Param("id")
	linesStr := c.DefaultQuery("lines", "100")
	
	lines, err := strconv.Atoi(linesStr)
	if err != nil {
		lines = 100
	}
	
	logEntries, err := h.dockerClient.GetContainerLogs(c.Request.Context(), containerID, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert LogEntry array to the format expected by frontend
	logs := make([]string, len(logEntries))
	for i, entry := range logEntries {
		logs[i] = fmt.Sprintf("[%s] %s", entry.Timestamp.Format("2006-01-02T15:04:05Z"), entry.Message)
	}

	response := map[string]interface{}{
		"logs":      logs,
		"timestamp": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, response)
}

func (h *ContainerHandler) GetContainerStats(c *gin.Context) {
	containerID := c.Param("id")
	
	stats, err := h.dockerClient.GetContainerStats(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ContainerHandler) PerformAction(c *gin.Context) {
	var action models.ContainerAction
	if err := c.ShouldBindJSON(&action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	switch action.Action {
	case "start":
		h.StartContainer(c)
	case "stop":
		h.StopContainer(c)
	case "restart":
		h.RestartContainer(c)
	case "remove":
		h.RemoveContainer(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
	}
}