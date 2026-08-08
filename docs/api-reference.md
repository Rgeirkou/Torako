# API Reference

The machine-readable [OpenAPI 3.0 specification](../api/openapi.yaml) is the source of truth for the API surface. This page is a human-readable companion.

## Base URL

```
https://api.example.com
```

## Authentication

All endpoints except `GET /stats` require an API key sent in the `X-API-Key` header:

```http
X-API-Key: your_api_key
```

API keys carry **scopes** (permission groups) and a **rank** (rate limit tier):

| Concept | Values                    | Description                                  |
| ------- | ------------------------- | -------------------------------------------- |
| Scope   | `tw`, `admin`             | `tw` — redemption endpoints; `admin` — key management |
| Rank    | `member`, `partner`, `admin` | Rate limit tier (60, 1000, unlimited req/window) |

## Endpoint Summary

| Method | Path                            | Auth  | Description                              |
| ------ | ------------------------------- | ----- | ---------------------------------------- |
| POST   | `/tw`                           | `tw`  | Redeem a TrueMoney gift (JSON body)      |
| GET    | `/tw/{phone}/{gift}`            | `tw`  | Redeem a TrueMoney gift (path params)    |
| POST   | `/keys`                         | admin | Create an API key                        |
| GET    | `/keys`                         | admin | List API keys                            |
| DELETE | `/keys/{id}`                    | admin | Revoke an API key                        |
| POST   | `/keys/{id}/rotate`             | admin | Rotate an API key                        |
| GET    | `/stats`                        | —     | All-time statistics (public)             |

## TrueMoney Wallet

### `POST /tw`

```bash
curl -X POST "https://api.example.com/tw" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key" \
  -d '{
    "phone": "0812345678",
    "gift": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }'
```

### `GET /tw/{phone}/{gift}`

```bash
curl "https://api.example.com/tw/0812345678/https%3A%2F%2Fgift.truemoney.com%2Fcampaign%2F%3Fv%3Dabc123" \
  -H "X-API-Key: your_api_key"
```

### Request Fields

| Field | Type   | Rules                                            |
| ----- | ------ | ------------------------------------------------ |
| `phone` | string | Thai mobile number, exactly 10 digits, starting with `0` |
| `gift`  | string | Gift code (20–60 alphanumeric characters) **or** a `https://` link on a trusted `truemoney.com` host |

### Response

`200 OK` with the raw upstream TrueMoney response passed through unchanged:

```json
{
  "data": {
    "status": { "code": "SUCCESS", "message": "SUCCESS" },
    "data": { "voucher": { "voucher_id": "1", "amount_baht": "100.00" } }
  }
}
```

## API Key Management

Key management endpoints require an API key with the `admin` scope. Keys are stored as SHA-256 hashes — the plaintext is returned **exactly once** (at creation or rotation) and is never persisted.

### `POST /keys` — Create

```bash
curl -X POST "https://api.example.com/keys" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_admin_key" \
  -d '{
    "name": "client-a",
    "rank": "partner",
    "scopes": ["tw"],
    "expires_at": "2027-01-01T00:00:00Z"
  }'
```

| Field        | Required | Rules                                          |
| ------------ | -------- | ---------------------------------------------- |
| `name`       | yes      | 1–64 characters                                |
| `scopes`     | yes      | At least one of `tw`, `admin`                  |
| `rank`       | no       | `member` (default), `partner`, `admin`         |
| `expires_at` | no       | RFC 3339 timestamp in the future               |

### `GET /keys` — List

```bash
curl "https://api.example.com/keys" -H "X-API-Key: your_admin_key"
```

### `DELETE /keys/{id}` — Revoke

```bash
curl -X DELETE "https://api.example.com/keys/1" -H "X-API-Key: your_admin_key"
```

Revoked keys are rejected by the auth middleware immediately.

### `POST /keys/{id}/rotate` — Rotate

```bash
curl -X POST "https://api.example.com/keys/1/rotate" -H "X-API-Key: your_admin_key"
```

Revokes the current key and issues a replacement with the same name, rank, and scopes. The replacement is created first, so a failed rotation leaves the old key usable.

## Statistics

### `GET /stats` — All-time statistics

Public endpoint — no API key required. Returns total amount (baht), successful operation count, failed attempts, and the TrueMoney channel breakdown:

```json
{
  "data": {
    "amount": 102.50,
    "count": 2,
    "errors": 0,
    "truemoney": { "amount": 102.50, "count": 2, "errors": 0 }
  }
}
```

Successful operations are deduplicated by upstream reference (each voucher is counted once); failed attempts are always counted.

## Response Envelope & Errors

Successful responses are wrapped in a `data` envelope; error responses use a consistent shape:

```json
{
  "error": "invalid input",
  "details": [
    { "field": "phone", "message": "must be a Thai phone number of exactly 10 digits" }
  ]
}
```

## HTTP Status Codes

| Status | Meaning                                                |
| ------ | ------------------------------------------------------ |
| `200 OK` | Success                                               |
| `201 Created` | API key created                                       |
| `204 No Content` | Key revoked                                      |
| `400 Bad Request` | Malformed JSON, unknown field, or invalid path param |
| `401 Unauthorized` | Missing or invalid API key                           |
| `403 Forbidden` | Revoked / expired key, or missing required scope      |
| `404 Not Found` | Upstream resource not found                           |
| `409 Conflict` | Duplicate API key, or the upstream reported the voucher as already consumed / out of stock |
| `422 Unprocessable Entity` | Input validation failed                      |
| `429 Too Many Requests` | Rate limit exceeded (honor `Retry-After`)          |
| `502 Bad Gateway` | Upstream unavailable or returned an invalid response  |
| `504 Gateway Timeout` | Upstream timed out                                   |
