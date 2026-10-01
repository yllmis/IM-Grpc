# task-mq consumption safeguards

`task-mq` uses Kafka as the durable buffer and applies an application-level
rate limiter immediately before message persistence. Kafka fetch batching and
this processing limit are intentionally separate.

## Production controls

See `etc/task-mq.production.yaml.example` for the complete safe example.

- `MessageProcessingRateLimit.GlobalMessagesPerSecond` is the total MongoDB
  write budget across the expected task-mq replicas.
- `ExpectedInstances` statically divides that budget. With a global budget of
  600 and two replicas, each process receives a 300 msg/s limiter.
- `Burst=1` keeps admission smooth. All chat readers in one process share the
  same limiter.
- `MessageRetry` retries a failed handler with exponential backoff. After the
  configured attempts, the original payload and error are synchronously
  published to the DLQ before the source offset is committed.
- The DLQ topic must be created before enabling retry.

## Lag and processing metrics

When `KafkaMonitoring.Enabled=true`, the process exposes `/metrics` and
`/healthz` on `KafkaMonitoring.ListenOn`. Important metrics are:

- `im_task_mq_consumer_lag{topic,group,partition}`
- `im_task_mq_messages_processed_total{status}`
- `im_task_mq_message_processing_seconds{status}`
- `im_task_mq_throttle_wait_seconds`
- `im_task_mq_message_retries_total`
- `im_task_mq_dead_letters_total`
- `im_task_mq_duplicate_messages_total`

Alert on sustained lag growth, not on a single short spike. A practical first
rule is lag growth for five minutes or a persist-latency SLO breach.

## Partitions and replicas

Kafka only assigns one active group member per partition. Before increasing
`Consumers` or deploying more task-mq replicas, increase the chat topic's
partition count and then set `ConsumerScaling.ExpectedPartitions`. Startup
refuses an undersized topic so scaling configuration cannot silently leave
readers idle.

Changing the rate, retry, monitoring, or scaling configuration currently
requires restarting task-mq. Runtime mutation of an existing limiter is not
implemented.

## Idempotency boundary

The stable `messageId` is used as MongoDB `_id`. Replays therefore do not
insert a second `chat_log` and do not increment the conversation summary a
second time. A duplicate may still re-run delivery so clients must de-duplicate
pushes by `messageId`.

MongoDB standalone mode cannot atomically insert `chat_log` and update the
separate `conversation` document. A crash exactly between those writes leaves
a reconciliation window. Eliminating that window requires a replica set plus
a transaction, or an asynchronous reconciliation job; the unique ID alone is
not an exactly-once guarantee.
