# Webhook Relay

A reliability layer for outgoing webhooks. Instead of every SaaS app
implementing its own retry queue, exponential backoff, HMAC signing, and
delivery tracking, it sends events to Webhook Relay and lets it handle
delivery.

```
Client → API → PostgreSQL → Redis Streams → Worker → Customer endpoint
```

## Features

- REST API for registering endpoints and sending events
- Asynchronous delivery via a Redis Streams queue + worker (crash-safe:
  abandoned deliveries are automatically reclaimed via `XAUTOCLAIM`)
- Retry policy with exponential backoff + jitter, up to 6 attempts, based on
  which HTTP status/error the destination returned
- Full delivery attempt history (status code, response body, duration, error)
- HMAC-SHA256 request signing (`Webhook-ID` / `Webhook-Timestamp` /
  `Webhook-Signature` headers) so receivers can verify authenticity
- A small dashboard (event list with status filters, per-event delivery
  history, manual retry)

## Status

Phases 1-8 of the build plan are complete: database, REST API, delivery,
queue + worker, retries, delivery history, signing, and dashboard. **Not yet
done: Phase 9 (security)** — there is no API authentication, rate limiting,
or SSRF protection on endpoint URLs yet, so this is not production-ready as
is. That's the next milestone.

## Stack

Go (`net/http`, no framework) · PostgreSQL · Redis Streams · React + Vite +
TypeScript for the dashboard · Docker Compose for local infra.

## Running it locally

Requires Docker, Go, and Node.js.

```powershell
powershell -ExecutionPolicy Bypass -File scripts\dev.ps1
```

This brings up Postgres + Redis, then opens the API, the worker, and the
dashboard dev server each in their own window.

- API: `http://localhost:8080`
- Dashboard: `http://localhost:5173`

### Try it

```powershell
$ep = Invoke-RestMethod http://localhost:8080/api/v1/endpoints -Method Post -ContentType "application/json" -Body (@{url="https://httpstat.us/200"} | ConvertTo-Json)

$body = @{ endpoint_id = $ep.id; event_type = "payment.completed"; payload = @{ payment_id = "pay_123"; amount = 4999 } } | ConvertTo-Json
Invoke-RestMethod http://localhost:8080/api/v1/events -Method Post -ContentType "application/json" -Body $body
```

Watch it show up and move to `DELIVERED` in the dashboard within a few
seconds.

## Project layout

```
cmd/api/       HTTP API server
cmd/worker/    Delivery worker (consumes the Redis Stream)
internal/      Application code (endpoints, events, delivery, queue, api)
migrations/    SQL migrations
dashboard/     React dashboard
```
