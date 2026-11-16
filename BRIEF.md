# Brief: webhook receiver

A small service that receives signed, Stripe-style payment webhooks and stores each event once.

## Constraints

- Standard library only: `net/http` for routing, `log/slog` for logging.
- Store events in memory, behind a `Store` interface, so the service runs with `go run .` and needs no setup.
- Listen on `:8080`.

## Endpoint

`POST /webhooks/{provider}` accepts a JSON event and stores it once, keyed by its `id`.

```json
{"id": "evt_1A2b3C4d5E6f7G8h9I0jK1lM", "type": "payment.succeeded", "data": {"amount": 2500, "currency": "gbp"}}
```

Event IDs must match `^evt_[A-Za-z0-9]{24}$`.

| Request | Response |
|---|---|
| New event | `202 Accepted` |
| Event already stored, including one that loses a race to be saved | `200 OK` |
| Invalid JSON or event ID | `400 Bad Request` |
| Body larger than 1 MiB | `400 Bad Request` |
| Missing or invalid signature | `401 Unauthorized` |
| Timestamp more than five minutes old | `401 Unauthorized` |

## Signatures

Every request carries a `Webhook-Signature` header, shown below:

```
Webhook-Signature: t=<unix timestamp in seconds>,v1=<signature>
```

The signature is the hex-encoded HMAC-SHA256 of `<timestamp>.<raw request body>`, using the shared secret as the key.

- Verify the signature in `net/http` middleware, before the handler runs.
- Sign the body exactly as it arrived, not a re-encoded version of the parsed JSON.
- Compare signatures with `hmac.Equal`.
- Reject timestamps more than five minutes old.
- Put the body back after reading it so the handler can decode it.

The secret comes from `WEBHOOK_SECRET`. When it's not set, use `dev-secret`.

## Errors

Error responses carry a short, fixed message. The details go to the logs.
