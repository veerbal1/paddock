paddock/
  collarsim/    ← simulated device fleet
  ingest/       ← MQTT → JetStream
  processor/    ← server-side geofence checks
  alerter/      ← outbox → Telegram
  api/          ← REST + WebSocket
  console/      ← React map