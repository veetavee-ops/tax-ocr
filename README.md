# Tax OCR System

Scaffolded project for the Tax OCR System defined in `CLAUDE.md`.

## Structure

- `frontend/liff` - LINE LIFF user application (React + Vite)
- `frontend/admin` - Admin dashboard (React + Vite + Tailwind)
- `backend` - Go HTTP API and service skeleton
- `database/migrations` - PostgreSQL migrations
- `infrastructure` - Local development infrastructure

## Prerequisites

- Node.js 18+ recommended for long-term maintenance
- npm 8+
- Go 1.22+ recommended
- Docker Desktop

Note: this workspace currently has Node.js 16 and no Go toolchain installed, so the scaffold is created without verifying the Go build.

## Quick Start

### Frontend apps

```powershell
Set-Location e:\tax-ocr\frontend\liff
npm install
npm run dev
```

```powershell
Set-Location e:\tax-ocr\frontend\admin
npm install
npm run dev
```

### Infrastructure

```powershell
Set-Location e:\tax-ocr\infrastructure
docker compose up -d
```

### Backend

Install Go first, then run:

```powershell
Set-Location e:\tax-ocr\backend
go run ./cmd
```
