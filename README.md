# Uptime Tracker

Sistema de monitoreo de APIs que verifica periódicamente la disponibilidad y latencia de servicios web. Usa workers concurrentes para ejecutar health checks, almacena historial en MongoDB, transmite resultados en vivo por WebSocket y envía alertas configurables.

## Arquitectura

```
Frontend (React) ◄── WS ──┐
                           │
                           ▼  REST
API (Go) ──► RabbitMQ ──► Workers ──► MongoDB
    │          │              │
    │          │              ▼
    ▼          ▼        Alert Worker ──► Slack/Email
  Redis     Scheduler
 (caché)   (ticker)
    │
    ▼
WebSocket Hub ──► Frontend
```

### Modos de ejecución

El binario acepta un flag `--mode` para ejecutar componentes de forma independiente:

- `--mode api` — HTTP server + WebSocket hub
- `--mode worker` — consume cola RabbitMQ, ejecuta checks
- `--mode scheduler` — consulta monitores activos y publica en cola
- `--mode all` — todo junto (desarrollo local)

## Stack

| Componente | Tecnología |
|------------|------------|
| Backend    | Go (chi/v5, gorilla/websocket, amqp091-go) |
| Frontend   | React + TypeScript + TailwindCSS + Recharts |
| Base de datos | MongoDB |
| Caché      | Redis |
| Mensajería | RabbitMQ |
| Contenedores | Docker + docker-compose |

## API REST

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check del sistema |
| POST | `/api/v1/auth/login` | Login (JWT) |
| POST | `/api/v1/auth/apikey` | Generar API key |
| POST | `/api/v1/monitors` | Crear monitor |
| GET | `/api/v1/monitors` | Listar monitores |
| GET | `/api/v1/monitors/:id` | Detalle de monitor |
| PUT | `/api/v1/monitors/:id` | Actualizar monitor |
| DELETE | `/api/v1/monitors/:id` | Eliminar monitor |
| POST | `/api/v1/monitors/:id/pause` | Pausar monitor |
| POST | `/api/v1/monitors/:id/resume` | Reanudar monitor |
| GET | `/api/v1/monitors/:id/history` | Historial de checks |
| GET | `/api/v1/monitors/:id/stats` | Estadísticas |
| GET | `/ws` | WebSocket (actualizaciones en vivo) |

## Setup local

### Requisitos

- Docker + docker-compose
- Go 1.22+ (para desarrollo)
- Node.js 20+ (para desarrollo)

### Con Docker

```bash
docker compose up --build
```

- Frontend: http://localhost:3000
- API: http://localhost:8080
- RabbitMQ UI: http://localhost:15672 (guest/guest)

### Desarrollo local

```bash
# Terminal 1: Infraestructura
docker compose up mongodb redis rabbitmq

# Terminal 2: Backend
cd backend
cp .env.example .env
go run ./cmd/uptimed --mode all

# Terminal 3: Frontend
cd frontend
npm install
npm run dev
```

### Variables de entorno

Ver `backend/.env.example` para todas las variables disponibles.

## Tests

```bash
# Backend
cd backend && go test ./... -v

# Frontend
cd frontend && npm test
```

## CI/CD

El proyecto incluye GitHub Actions que ejecuta:

- Lint (golangci-lint + ESLint)
- Tests (Go + Vitest)
- Build de imágenes Docker
