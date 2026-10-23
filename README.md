# 🔗 URL Shortener

A full-stack URL shortener built with **Go, Gin, PostgreSQL, Redis, React, and Vite**.

The application lets users create and manage short links, then redirect visitors to their original destinations. It includes authentication, caching, rate limiting, and URL safety checks.

**Project status:** Work in progress / portfolio project.

## ✨ Features

- **URL management** — create, list, retrieve, and delete your short links.
- **Authentication** — account signup, login, logout, and session handling.
- **Fast redirects** — Redis caching backed by PostgreSQL persistence.
- **URL safety** — HTTP/HTTPS validation and malicious-domain checks.
- **API protection** — request-size limits, rate limiting, and security headers.

## 🧭 Architecture

```text
React + Vite frontend
        │
        ▼
     Go / Gin API
        │
   ┌────┴─────┐
   ▼          ▼
PostgreSQL   Redis
(users and   (cached URL
 URLs)        lookups)
```

PostgreSQL is the persistent data store. Redis caches URL lookups to reduce repeated database reads during redirects.

## 🛠️ Tech stack

| Technology | Purpose |
|---|---|
| Go, Gin | Backend and HTTP API |
| PostgreSQL, pgx | Persistent storage |
| Redis, go-redis | URL caching |
| React, Vite | Frontend |
| Go testing, ESLint | Testing and code quality |

## 🚀 Getting started

### Requirements

- Go version specified in `go.mod`
- Node.js and npm
- PostgreSQL
- Redis

### 1. Configure the backend

```bash
cp .env.example .env
cp blocklist.example.txt blocklist.txt
```

Edit `.env` with your local PostgreSQL and Redis settings. Create the database specified by `DBName`, then apply the SQL files in `migrations/` in filename order.

### 2. Run the backend

```bash
go run .
```

The example configuration uses `http://localhost:8080`.

### 3. Run the frontend

In a separate terminal:

```bash
cd frontend
npm install
cp .env.example .env.development
npm run dev
```

Open the local address printed by Vite.

## ⚙️ Configuration

Backend settings are documented in `.env.example`. The main settings cover:

- Server host, port, and public short-link base URL
- PostgreSQL connection
- Redis connection and cache TTL
- Optional Google Safe Browsing integration and local domain blocklist

The frontend's `.env.example` contains `VITE_SHORT_BASE`, used when displaying generated short links. Never commit real credentials or your `.env` file.

## 🧪 Development checks

From the repository root:

```bash
go test ./...
```

From `frontend/`:

```bash
npm run lint
npm run build
```

## ⚠️ Current status

The project is under development. Verify the setup, tests, and security configuration in your environment before deploying it publicly.
