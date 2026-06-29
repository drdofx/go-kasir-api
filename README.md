# go-kasir-api

Backend API for a POS/cashier application built with Go, PostgreSQL, JWT auth, database migrations, and Docker.

## Stack

- Go `net/http`
- PostgreSQL
- `database/sql` + `lib/pq`
- `golang-migrate`
- JWT auth
- Zerolog request logging

## Local Run

```bash
cp .env.example .env
docker compose up --build
```

The API runs on `http://localhost:8080` by default.

Health check:

```bash
curl http://localhost:8080/health
```

The development seed creates this user when the database is empty:

- Username: `kasir`
- Password: value of `ADMIN_PASSWORD`

## Configuration

Required:

- `DB_CONN`: PostgreSQL connection string.
- `JWT_SECRET`: secret used to sign JWTs. Use a long random value, at least 32 characters in production.
- `ADMIN_PASSWORD`: initial admin password when the database is empty.

Common:

- `APP_ENV`: `development` or `production`.
- `PORT`: port HTTP, default `8080`.
- `CORS_ALLOWED_ORIGIN`: comma-separated allowed origins.
- `AUTO_MIGRATE`: run migrations at startup, default `true`.
- `MIGRATIONS_PATH`: path folder migration, default `migrations`.
- `SERVER_READ_TIMEOUT`: default `10s`.
- `SERVER_READ_HEADER_TIMEOUT`: default `5s`.
- `SERVER_WRITE_TIMEOUT`: default `30s`.
- `SERVER_IDLE_TIMEOUT`: default `120s`.
- `SERVER_SHUTDOWN_TIMEOUT`: default `10s`.
- `MAX_REQUEST_BODY_BYTES`: global request body limit, default `1048576`.
- `ACCESS_TOKEN_TTL`: JWT access token lifetime, default `15m`.
- `REFRESH_TOKEN_TTL`: refresh token lifetime, default `720h`.

## Production Notes

For production:

- Set `APP_ENV=production`.
- Set a unique, long `JWT_SECRET`.
- Set `ADMIN_PASSWORD`; do not use the development default.
- Consider `AUTO_MIGRATE=false` and run migrations as a separate release step.
- Set `CORS_ALLOWED_ORIGIN` only to valid frontend origins.
- Run behind a reverse proxy or load balancer that terminates TLS.

## Tests

```bash
go test ./...
```

## API Surface

The primary endpoints are available under `/api/v1`, including:

- `/api/v1/auth/login`
- `/api/v1/auth/refresh`
- `/api/v1/auth/me`
- `/api/v1/products`
- `/api/v1/categories`
- `/api/v1/checkout`
- `/api/v1/transactions`
- `/api/v1/customers`
- `/api/v1/payment-types`
- `/api/v1/returns`
- `/api/v1/suppliers`
- `/api/v1/purchase-orders`
- `/api/v1/inventory/alerts`
- `/api/v1/branches`
- `/api/v1/report`
- `/api/v1/report/today`

Legacy `/api/...` endpoints are still available for selected resources.

## Pagination

List endpoints accept `page` and `per_page` query parameters. Pagination metadata is returned in response headers:

- `X-Page`
- `X-Per-Page`
- `X-Total-Count`
