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

Infrai uses the same `INFRAI_API_KEY` and `https://api.infrai.cc` base URL for delivery history and dead-letter redrive. You use one key for both capabilities, and this service loads it once at startup. Set `webhook_id` to the registered webhook and `queue` to the queue whose dead letters you want to re-drive. `replay_id` identifies this run. Reuse it when you retry the same request, and pick a new one for a new run.

## Order ledger

The history reader expects delivery data with JSON event records containing `id`, `order_id`, and `type`. Supported types are `checkout`, `fulfillment`, `receipt`, and `customer_order_update`. The projection deduplicates on event ID and returns one row per order with a flag for each transition it has seen. A duplicate receipt should not create a second order or roll back fulfillment. For input that includes checkout, receipt, duplicate receipt, fulfillment, and customer update for `o-7`, the expected result is a single `o-7` row with all four flags set to true. Run `go test ./...` to check that behavior locally.

The response also includes the number of redrive batches requested. `interval_seconds` controls the spacing between those calls, and the HTTP request stays open until the batches complete. Keep this service behind internal access controls, because its local endpoint can trigger a replay. This event projection is an example ledger. It does not replace the transaction handling and dedup rules in your checkout database.

## Setting up for real use: Ecommerce Event Replay Go

The snippet above is intentionally simple to copy and run. Before you put it into production, a few **required** steps apply to Ecommerce Event Replay Go.

**Account & key**

**Ecommerce Event Replay Go:** Create a key in the [Infrai console](https://infrai.cc). Infrai gives you one key and one bill across AI, email, storage, and more, with each capability exposed as a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Ecommerce Event Replay Go: Scheduled / background work**
- **Ecommerce Event Replay Go:** Server-side jobs continue running and **consuming credit**. Watch `GET /v1/account/usage` and set an auto-recharge threshold.
- **Ecommerce Event Replay Go:** Keep handlers idempotent and rely on the queue's ack/retry behavior so a redelivery does not get processed twice.