# Contexto del proyecto — Uptime Tracker

Sistema auto-hospedable de monitoreo de disponibilidad de APIs/servicios web (estilo Uptime Kuma / Pingdom).
Registra monitores (URL + método + intervalo), ejecuta health checks concurrentes, guarda historial,
transmite resultados en vivo por WebSocket y envía alertas por email/webhook.

Este archivo es la fuente de contexto para agentes de IA. Mantenerlo actualizado cuando cambien
la arquitectura, los comandos, los endpoints o el estado de las funcionalidades.

## Stack

| Componente   | Tecnología                                                        |
|--------------|-------------------------------------------------------------------|
| Backend      | Go 1.22, chi/v5, gorilla/websocket, amqp091-go, zerolog, JWT v5   |
| Frontend     | React 18 + TypeScript + Vite + TailwindCSS + Recharts + Axios     |
| Base de datos| MongoDB 7 (`monitors`, `check_results`)                           |
| Caché        | Redis 7 (último resultado, contador de fallos, rate limit)        |
| Mensajería   | RabbitMQ 3.13 (exchange topic `uptime.checks`)                    |
| Infra        | Docker + docker-compose, nginx para el frontend                   |
| CI           | GitHub Actions: golangci-lint, go test -race, ESLint, Vitest, build Docker |

Módulo Go: `github.com/lv99/uptime-tracker`.

## Estructura del repositorio

```
backend/
  cmd/uptimed/main.go            # entrypoint, flag --mode (api|worker|scheduler|all)
  internal/config/               # carga de env vars (.env con godotenv)
  internal/model/                # Monitor, CheckResult, CheckJob, AlertMessage, Stats
  internal/repository/           # acceso a MongoDB (monitor_repo, check_repo)
  internal/api/                  # router chi, handlers, auth JWT/API key, middleware
  internal/scheduler/            # ticker que encola checks vencidos
  internal/worker/               # pool de workers que ejecuta los checks HTTP
  internal/alerting/             # consume alertas y envía email SMTP / webhook
  internal/messaging/            # conexión RabbitMQ, declaración de colas, publish/consume
  internal/ws/                   # hub WebSocket (broadcast de resultados)
  Dockerfile
frontend/
  src/pages/                     # Login, Dashboard, MonitorDetail, Settings
  src/components/                # Navbar, MonitorCard, MonitorForm, LatencyChart
  src/hooks/                     # useMonitors, useWebSocket
  src/services/api.ts            # cliente Axios + interceptor de auth
  src/types/index.ts             # tipos compartidos (espejo de los JSON del backend)
  nginx.conf, vite.config.ts, Dockerfile
test-api/main.go                 # servidor de pruebas: /ok, /slow, /flaky, /error, /status/N, /echo
docker-compose.yml               # mongodb, redis, rabbitmq, api, frontend, test-api
.github/workflows/ci.yml
```

## Flujo de un check

1. **Scheduler** (tick cada `SCHEDULER_INTERVAL`, 10s por defecto) lista monitores `active` y publica
   `check.pending` si `now - último check >= interval_s` (o si nunca se chequeó).
2. **Worker pool** (`WORKER_COUNT`, 5 por defecto) consume `checks.pending`, hace la petición HTTP
   con timeout/método/headers del monitor. Éxito = status `200–399`.
3. Guarda en `check_results`, cachea en Redis y publica `check.result`.
4. El **API** consume `checks.results` y reenvía por WebSocket `/ws` (mensaje `CheckResultMessage`).
5. Si falla: incrementa `monitor:<id>:fail_count`; publica `check.alert` en los fallos 1, 3, 7, 15, 31…
   (`count == 1 || count == 3 || (count > 3 && ((count+1)&count) == 0)`).
6. **Alert worker** consume `checks.alerts`, busca el monitor y envía email (SMTP) y/o webhook genérico
   con payload `{"subject": "...", "body": "..."}`.

## Comandos

```bash
# Todo con Docker (modo all dentro del contenedor api)
docker compose up --build
# Frontend http://localhost:3000 · API http://localhost:8080
# RabbitMQ UI http://localhost:15672 (guest/guest) · test-api http://localhost:9000

# Infra solamente
docker compose up mongodb redis rabbitmq

# Backend
cd backend && cp .env.example .env && go run ./cmd/uptimed --mode all
go test ./... -v -race -count=1
gofmt -w . && go vet ./...

# Frontend
cd frontend && npm install && npm run dev     # puerto 3000, proxy a :8080
npm run lint
npm test -- --run                              # sin --run se queda en watch mode
npm run build                                  # tsc -b && vite build
```

Variables de entorno del backend: ver `backend/.env.example` (PORT, MONGO_URI, MONGO_DB, REDIS_ADDR,
REDIS_PASSWORD, RABBITMQ_URI, JWT_SECRET, SCHEDULER_INTERVAL, WORKER_COUNT, LOG_LEVEL, SMTP_*).

## API REST

Base: `/api/v1`. Los `{id}` son ObjectID hex de Mongo.

| Método | Ruta                          | Auth | Descripción                        |
|--------|-------------------------------|------|------------------------------------|
| GET    | `/health`                     | No   | Health check del API               |
| POST   | `/auth/login`                 | No   | Devuelve JWT (24h)                 |
| POST   | `/auth/apikey`                | No   | Genera API key `ut_<64 hex>`       |
| POST   | `/monitors`                   | Sí   | Crear monitor                      |
| GET    | `/monitors`                   | Sí   | Listar con `last_result` y `uptime_24h` |
| GET    | `/monitors/{id}`              | Sí   | Detalle + último resultado + uptime |
| PUT    | `/monitors/{id}`              | Sí   | Actualizar (reemplazo completo)    |
| DELETE | `/monitors/{id}`              | Sí   | Eliminar                           |
| POST   | `/monitors/{id}/pause`        | Sí   | status → `paused`                  |
| POST   | `/monitors/{id}/resume`       | Sí   | status → `active`                  |
| GET    | `/monitors/{id}/history`      | Sí   | `?limit` (máx 100, def 50) `&offset` |
| GET    | `/monitors/{id}/stats`        | Sí   | `?period=24h\|7d\|30d`             |
| GET    | `/ws`                         | No   | WebSocket de resultados en vivo    |

Auth: `Authorization: Bearer <jwt>` o `X-API-Key: <key>`. Rate limit: 60 req/60s por IP+ruta (Redis).

## Almacenamiento

MongoDB (`uptime_tracker`):
- `monitors`: `_id`, `name`, `url`, `method`, `interval_s` (def 60), `timeout_ms` (def 5000), `status`
  (`active|paused`), `headers`, `alert_email`, `alert_webhook`, `created_at`, `updated_at`.
- `check_results`: `_id`, `monitor_id`, `status_code`, `latency_ms`, `success`, `error_msg`, `checked_at`.

Redis:
- `monitor:<id>:last_result` → JSON del último `CheckResult` (TTL 1h).
- `monitor:<id>:fail_count` → contador de fallos consecutivos (TTL 24h desde el primer fallo).
- `ratelimit:<ip>:<path>` → contador (TTL = ventana).

RabbitMQ: exchange durable `uptime.checks` (topic) con colas durables `checks.pending`,
`checks.results`, `checks.alerts` y routing keys `check.pending`, `check.result`, `check.alert`.

## Frontend

Rutas: `/login`, `/` (dashboard), `/monitors/:id`, `/settings` (protegidas por presencia de
`token` o `apiKey` en localStorage). El dashboard agrupa tarjetas en Down / Operational / Pending / Paused.
El detalle muestra métricas, gráfico de latencia y tabla de historial. El WebSocket se conecta a
`ws(s)://<host>/ws` y reconecta a los 3s.

## Convenciones

- Go: `gofmt`, paquetes por responsabilidad, errores envueltos con `%w`, logging estructurado con
  zerolog (`.Str()`, `.Err()`, `.Msg()`), JSON en `snake_case`, respuestas de error `{"error": "..."}`.
- Frontend: componentes funcionales con `export default`, hooks con `export function`, estilos solo
  con clases Tailwind (sin CSS modules), llamadas HTTP centralizadas en `src/services/api.ts`,
  tipos en `src/types/index.ts` (mantener sincronizados con los JSON del backend).
- No commitear artefactos: `backend/uptimed`, `frontend/dist/`, `node_modules/`, `.env`.
- Al cambiar endpoints o comportamiento, actualizar `README.md` y este `AGENTS.md`.
- Añadir tests junto al código que se modifica (Go: `_test.go` en el mismo paquete; frontend: Vitest).

## Estado actual y deuda técnica conocida

Implementado y funcional de punta a punta: CRUD de monitores, scheduler, workers, persistencia,
stats (uptime, media, p95, p99), caché, rate limit, WebSocket, alertas email/webhook, Docker y CI.

Limitaciones conocidas (no "arreglar" sin pedirlo, pero tenerlas en cuenta):

- **Auth de prototipo**: el login acepta cualquier usuario/contraseña no vacíos; no hay usuarios ni
  hash de passwords. Las API keys se generan pero no se persisten ni validan: cualquier `X-API-Key`
  pasa. `middleware.go` hace `apiKey[:8]`, que entra en pánico con keys de menos de 8 caracteres.
  El endpoint `/ws` no tiene auth.
- **Broadcast WS roto**: en `main.go` (`consumeResults`) se pasa `msg.Body` (`[]byte`) a
  `hub.Broadcast`, que hace `json.Marshal` y produce un string base64 en vez del objeto JSON.
  El fix correcto es `hub.Broadcast(json.RawMessage(msg.Body))`.
- **Alert worker solo corre en `--mode all`**; no existe `--mode alert`. No hay notificación de
  recuperación ni formato específico de Slack.
- **Escalado**: el scheduler no usa lock distribuido (instancias duplicarían checks); varias
  instancias del API compiten por `checks.results`, por lo que el broadcast WS no llega a todos.
- **Rendimiento**: no se crean índices en Mongo; `List` hace N+1 consultas (último resultado +
  stats 24h por monitor); los percentiles se calculan en Go tras traer todas las latencias.
- **Concurrencia WS**: `Hub.Broadcast` borra clientes del map bajo `RLock` (debería ser `Lock`) y
  puede cerrar `client.send` dos veces.
- **Tests**: frontend solo tiene un test placeholder; backend no cubre workers, scheduler, alerting
  ni repositorios.
- **UI**: el selector 24h/7d/30d solo afecta a las stats, no filtra el historial del gráfico.
  `PUT /monitors/{id}` sobrescribe con ceros si el body es parcial.
- Sin OpenAPI/Swagger, multi-usuario, retención de historial ni métricas Prometheus.

## Reglas para agentes

1. Antes de cambiar algo, leer el archivo relevante; no asumir que el README refleja el estado real.
2. Verificar siempre con `go test ./...` y `npm run lint` (y `npm run build` si se toca TS).
3. No introducir secretos reales; usar `backend/.env.example` como plantilla.
4. No cambiar el esquema de MongoDB ni las routing keys de RabbitMQ sin migración/coordinación.
5. Los bugs de la sección anterior son deuda conocida: si se corrigen, actualizar esta sección.
6. Mantener el estilo existente (zerolog, chi, Tailwind, Axios service layer) en lugar de añadir
   nuevas librerías o patrones sin necesidad.
