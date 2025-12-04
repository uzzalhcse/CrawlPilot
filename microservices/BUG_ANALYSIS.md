# CRITICAL BUG FOUND

## Issue: Worker Not Receiving Pub/Sub Messages

### Evidence:
1. **Orchestrator**: Publishing tasks successfully (`Published batch of tasks {"count": 1}`)
2. **Worker**: Subscription configured but NO messages received
3. **Stats Updated**: Somehow stats are received, suggesting phantom worker OR direct DB update

### Root Cause Analysis:

The worker says:
```
Starting to receive messages from subscription {"subscription": "crawlify-tasks-sub"}
Subscription configuration {"max_outstanding": 100, "num_goroutines": 10}
```

But **NEVER** logs "📨 Message received from Pub/Sub" - meaning `sub.Receive()` is blocking but not calling the callback.

### Possible Causes:

1. **Pub/Sub Emulator Issue**: Messages published but not delivered to subscription
2. **Go Pub/Sub Client Bug**: Receive() starts but doesn't pull messages
3. **Context Cancelled**: Worker's context cancelled before receiving
4. **Subscription ACK Issue**: Messages delivered but immediately re-queued

### Next Steps:

1. ✅ Check if messages exist in topic (via pull API)
2. ✅ Verify subscription has messages
3. ❌ Test manual pull to confirm messages exist
4. ❌ Check worker's context lifecycle

### Test Commands:

```bash
# Check messages in subscription
curl -X POST http://localhost:8085/v1/projects/crawlify-local/subscriptions/crawlify-tasks-sub:pull \
  -H "Content-Type: application/json" \
  -d '{"maxMessages": 10}'

# Check topic
curl http://localhost:8085/v1/projects/crawlify-local/topics/crawlify-tasks
```

### Status: INVESTIGATING
