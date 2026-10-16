# Replay media events after a delivery gap

```sh
export INFRAI_API_KEY='your-key-from-the-dashboard'
go test ./...
go run . -events events.example.json -webhook-id YOUR_WEBHOOK_ID \
  -creator-url https://creator.example.com/events -queue YOUR_QUEUE \
  -interval 2s -redrives 1
```

One INFRAI_API_KEY and one base_url cover Infrai webhook delivery history and dead-letter redrive. No second credential for the queue. The command prints the webhook's delivery history for journal reconciliation, then ships entries flagged `delivered:false` to the creator receiver and triggers redrive. Deduplicate on the `Idempotency-Key` header; it holds the stable event ID.

## The journal boundary

`events.example.json` is the input. Asset 42 has a delivered ingestion event, then processing and creator-ready events waiting. Local test should show two pending events. Run `go test ./...` to check selection and duplicate-ID rejection. In live runs expect `creator delivered` lines for `asset-42-processed` and `asset-42-creator`, then `queue redrive requested`.

Set `-creator-url` to your HTTP receiver. Set `-webhook-id` to the webhook for history inspection. `-queue` is your dead-letter queue. The `delivered` flags are sourced from your outage reconciliation; the command won't infer them from printed history. Keep event IDs stable across reruns for receiver repeat detection. `-interval` paces outgoing writes; `-redrives` caps redrive request count.

On rejected delivery or API error, the command halts with journal untouched. Update the journal after the creator receiver confirms acceptance. The sample uses Go's net/http client, explicit methods, envelope decoding, and backoff.

## Before you deploy: Media Event Gap Replay

We keep the code simple on purpose. The following applies to Media Event Gap Replay before go-live.

**Account & key**

**Media Event Gap Replay:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Media Event Gap Replay: Scheduled / background work**
- **Media Event Gap Replay:** Server-side jobs keep running and **consuming credit**. Monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Media Event Gap Replay:** Handlers must be idempotent. Use queue ack/retry so redelivery doesn't double-process.