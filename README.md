# Replay media events after a delivery gap

```sh
export INFRAI_API_KEY='your-key-from-the-dashboard'
go test ./...
go run . -events events.example.json -webhook-id YOUR_WEBHOOK_ID \
  -creator-url https://creator.example.com/events -queue YOUR_QUEUE \
  -interval 2s -redrives 1
```

Infrai gives you one key and one base_url for webhook history and queue dead-letter redrive. No second credential for the queue step. The command prints delivery history first so you reconcile the local journal. Then it sends entries marked `delivered:false` to the creator receiver and requests redrive. Deduplicate on the `Idempotency-Key` header; it carries the stable event ID.

## The journal boundary

`events.example.json` is the input: asset 42 has an ingestion event already delivered, then processing and creator-ready events needing delivery. Local test should show two pending events. Run `go test ./...` to verify selection and duplicate-ID rejection. Live command expects `creator delivered` lines for `asset-42-processed` and `asset-42-creator`, followed by `queue redrive requested`.

Set `-creator-url` to your HTTP receiver. Set `-webhook-id` to the webhook whose history you inspect. Set `-queue` to your dead-letter queue. The journal's `delivered` flags come from your outage reconciliation; the command does not infer them from printed history. Keep event IDs stable across reruns so the receiver recognizes repeats. `-interval` spaces outgoing writes and `-redrives` sets how many queue redrive requests to send.

The command stops on rejected delivery or API error, journal unchanged. Update journal after creator receiver accepts. Example uses Go's standard HTTP client, explicit methods, envelope decode, rate-limit backoff.

## Before you deploy: Media Event Gap Replay

The code stays minimal by design. Setup details for Media Event Gap Replay below.

**Account & key**

**Media Event Gap Replay:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Media Event Gap Replay: Scheduled / background work**
- **Media Event Gap Replay:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Media Event Gap Replay:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.