# Docker GUI

A desktop application for managing Docker containers with a modern web interface. Built with Go backend and Next.js/Tauri frontend.

## Features

- View all Docker containers
- Start, stop, and restart containers
- Remove containers
- View container logs and statistics
- Real-time monitoring
- Activity logging with Turso database

## Prerequisites

- Docker installed and running
- Go 1.19+
- Node.js 18+
- Tauri prerequisites (for desktop app)

## Quick Start

### Run the complete application

### Backend only
```bash
cd backend
go mod tidy
```

### Frontend only
```bash
cd frontend
npm install
npm run dev          # Web development
npm run tauri:dev    # Desktop app development
```

## Build for Production

```bash
cd frontend
npm run build
npm run tauri:build  # Creates desktop app
```

## Architecture

- **Backend**: Go REST API with Gin framework
- **Frontend**: Next.js with Tauri for desktop integration
- **Database**: Turso (LibSQL) for activity logging
- **Docker**: Direct Docker API communication

## API Endpoints

- `GET /api/v1/containers` - List containers
- `POST /api/v1/containers/:id/start` - Start container
- `POST /api/v1/containers/:id/stop` - Stop container
- `POST /api/v1/containers/:id/restart` - Restart container
- `DELETE /api/v1/containers/:id` - Remove container
- `GET /api/v1/containers/:id/logs` - Container logs
- `GET /api/v1/containers/:id/stats` - Container statistics
- `GET /api/v1/logs` - Activity logs