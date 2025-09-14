package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type DB struct {
	conn *sql.DB
}

func NewDatabase() (*DB, error) {
	dbURL := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	if dbURL == "" || authToken == "" {
		return nil, fmt.Errorf("TURSO_DATABASE_URL and TURSO_AUTH_TOKEN environment variables are required")
	}

	fullURL := dbURL + "?authToken=" + authToken

	db, err := sql.Open("libsql", fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Turso database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping Turso database: %w", err)
	}

	log.Println("Successfully connected to Turso database")

	database := &DB{conn: db}

	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return database, nil
}

func (db *DB) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS container_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			container_id TEXT NOT NULL,
			container_name TEXT NOT NULL,
			action TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			user_info TEXT,
			details TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS container_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			container_id TEXT NOT NULL,
			container_name TEXT NOT NULL,
			cpu_usage REAL,
			memory_usage REAL,
			memory_limit REAL,
			network_rx REAL,
			network_tx REAL,
			disk_read REAL,
			disk_write REAL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			total_containers INTEGER,
			running_containers INTEGER,
			stopped_containers INTEGER,
			paused_containers INTEGER,
			total_cpu_usage REAL,
			total_memory_usage REAL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_container_logs_container_id ON container_logs(container_id)`,
		`CREATE INDEX IF NOT EXISTS idx_container_logs_timestamp ON container_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_container_metrics_container_id ON container_metrics(container_id)`,
		`CREATE INDEX IF NOT EXISTS idx_container_metrics_timestamp ON container_metrics(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_system_metrics_timestamp ON system_metrics(timestamp)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) LogContainerAction(containerID, containerName, action, userInfo, details string) error {
	query := `
	INSERT INTO container_logs (container_id, container_name, action, user_info, details)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query, containerID, containerName, action, userInfo, details)
	return err
}

func (db *DB) StoreContainerMetrics(containerID, containerName string, cpuUsage, memoryUsage, memoryLimit, networkRx, networkTx, diskRead, diskWrite float64) error {
	query := `
	INSERT INTO container_metrics 
	(container_id, container_name, cpu_usage, memory_usage, memory_limit, network_rx, network_tx, disk_read, disk_write)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query, containerID, containerName, cpuUsage, memoryUsage, memoryLimit, networkRx, networkTx, diskRead, diskWrite)
	return err
}

func (db *DB) StoreSystemMetrics(totalContainers, runningContainers, stoppedContainers, pausedContainers int, totalCpuUsage, totalMemoryUsage float64) error {
	query := `
	INSERT INTO system_metrics 
	(total_containers, running_containers, stopped_containers, paused_containers, total_cpu_usage, total_memory_usage)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query, totalContainers, runningContainers, stoppedContainers, pausedContainers, totalCpuUsage, totalMemoryUsage)
	return err
}

func (db *DB) GetContainerLogs(containerID string, limit int) ([]ContainerLog, error) {
	query := `
	SELECT id, container_id, container_name, action, timestamp, user_info, details
	FROM container_logs
	WHERE container_id = ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := db.conn.Query(query, containerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ContainerLog
	for rows.Next() {
		var log ContainerLog
		err := rows.Scan(
			&log.ID,
			&log.ContainerID,
			&log.ContainerName,
			&log.Action,
			&log.Timestamp,
			&log.UserInfo,
			&log.Details,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func (db *DB) GetAllLogs(limit int) ([]ContainerLog, error) {
	query := `
	SELECT id, container_id, container_name, action, timestamp, user_info, details
	FROM container_logs
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ContainerLog
	for rows.Next() {
		var log ContainerLog
		err := rows.Scan(
			&log.ID,
			&log.ContainerID,
			&log.ContainerName,
			&log.Action,
			&log.Timestamp,
			&log.UserInfo,
			&log.Details,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func (db *DB) GetContainerMetrics(containerID string, hours int) ([]ContainerMetric, error) {
	query := `
	SELECT container_id, container_name, cpu_usage, memory_usage, memory_limit, 
	       network_rx, network_tx, disk_read, disk_write, timestamp
	FROM container_metrics
	WHERE container_id = ? AND timestamp > datetime('now', '-' || ? || ' hours')
	ORDER BY timestamp DESC
	`

	rows, err := db.conn.Query(query, containerID, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []ContainerMetric
	for rows.Next() {
		var metric ContainerMetric
		err := rows.Scan(
			&metric.ContainerID,
			&metric.ContainerName,
			&metric.CPUUsage,
			&metric.MemoryUsage,
			&metric.MemoryLimit,
			&metric.NetworkRx,
			&metric.NetworkTx,
			&metric.DiskRead,
			&metric.DiskWrite,
			&metric.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (db *DB) GetSystemMetrics(hours int) ([]SystemMetric, error) {
	query := `
	SELECT total_containers, running_containers, stopped_containers, paused_containers,
	       total_cpu_usage, total_memory_usage, timestamp
	FROM system_metrics
	WHERE timestamp > datetime('now', '-' || ? || ' hours')
	ORDER BY timestamp DESC
	`

	rows, err := db.conn.Query(query, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []SystemMetric
	for rows.Next() {
		var metric SystemMetric
		err := rows.Scan(
			&metric.TotalContainers,
			&metric.RunningContainers,
			&metric.StoppedContainers,
			&metric.PausedContainers,
			&metric.TotalCPUUsage,
			&metric.TotalMemoryUsage,
			&metric.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

type ContainerLog struct {
	ID            int    `json:"id"`
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Action        string `json:"action"`
	Timestamp     string `json:"timestamp"`
	UserInfo      string `json:"user_info"`
	Details       string `json:"details"`
}

type ContainerMetric struct {
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	MemoryLimit   float64   `json:"memory_limit"`
	NetworkRx     float64   `json:"network_rx"`
	NetworkTx     float64   `json:"network_tx"`
	DiskRead      float64   `json:"disk_read"`
	DiskWrite     float64   `json:"disk_write"`
	Timestamp     time.Time `json:"timestamp"`
}

type SystemMetric struct {
	TotalContainers   int       `json:"total_containers"`
	RunningContainers int       `json:"running_containers"`
	StoppedContainers int       `json:"stopped_containers"`
	PausedContainers  int       `json:"paused_containers"`
	TotalCPUUsage     float64   `json:"total_cpu_usage"`
	TotalMemoryUsage  float64   `json:"total_memory_usage"`
	Timestamp         time.Time `json:"timestamp"`
}

func (db *DB) Close() error {
	return db.conn.Close()
}