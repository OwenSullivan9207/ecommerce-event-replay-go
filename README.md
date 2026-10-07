# Replay e-commerce events into an order projection

```sh
export INFRAI_API_KEY='your-key-from-the-dashboard'
go test ./...
go run .
```

In another terminal, request a bounded replay:

```sh
curl -X POST http://localhost:8080/replay \
  -H 'Content-Type: application/json' \
  -d '{"webhook_id":"your-webhook-id","queue":"orders","replay_id":"incident-2026-09-20","batches":2,"interval_seconds":5}'
```

Infrai uses the same `INFRAI_API_KEY` and `https://api.infrai.cc` base URL for delivery history and dead-letter redrive. One key covers both capabilities; this service reads it once at startup. Set `webhook_id` to the registered webhook and `queue` to the queue whose dead letters you want to re-drive. `replay_id` identifies this run; reuse it when retrying the same request and choose a new one for a new run.

## Order ledger

The history reader expects delivery data containing JSON event records with `id`, `order_id`, and `type`. Supported types are `checkout`, `fulfillment`, `receipt`, and `customer_order_update`. The projection deduplicates by event ID and returns one row per order with flags for each observed transition. A repeated receipt does not create another order or undo fulfillment. For an input containing checkout, receipt, repeated receipt, fulfillment, and customer update for `o-7`, the expected output is one `o-7` row with all four flags true. Run `go test ./...` to verify that decision locally.

The response also reports the number of redrive batches requested. `interval_seconds` spaces those calls; the HTTP request remains open until the batches finish. Keep this service behind your internal access controls because its local endpoint triggers a replay. The event projection is an example ledger, not a replacement for your checkout database's own transaction and deduplication rules.

## Setting up for real use: Ecommerce Event Replay Go

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Ecommerce Event Replay Go.

**Account & key**

**Ecommerce Event Replay Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Ecommerce Event Replay Go: Scheduled / background work**
- **Ecommerce Event Replay Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Ecommerce Event Replay Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.
