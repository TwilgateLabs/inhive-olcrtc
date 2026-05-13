# Configuration

olcrtc reads its entire runtime configuration from a single YAML file.
There are no other CLI flags.

```bash
olcrtc /etc/olcrtc/server.yaml
```

Examples:

- [`server.example.yaml`](./server.example.yaml)
- [`client.example.yaml`](./client.example.yaml)

## Schema

| YAML path                                                        | Notes                                                     |
|------------------------------------------------------------------|-----------------------------------------------------------|
| `mode`                                                           | `srv`, `cnc`, or `gen`                                    |
| `link`                                                           | `direct`                                                  |
| `auth.provider`                                                  | `telemost`, `jazz`, `wbstream`, `none`                    |
| `room.id`                                                        | conference room id                                        |
| `room.client_id`                                                 | deprecated, will be removed                               |
| `crypto.key`                                                     | 64-char hex (32 bytes)                                    |
| `net.transport`                                                  | `datachannel`, `videochannel`, `seichannel`, `vp8channel` |
| `net.dns`                                                        | resolver `host:port`                                      |
| `socks.host` / `.port`                                           | client-side listener                                      |
| `socks.user` / `.pass`                                           | optional client-side auth                                 |
| `socks.proxy_addr` / `.proxy_port`                               | server-side egress proxy                                  |
| `engine.name` / `.url` / `.token`                                | only when `auth.provider: none`                           |
| `video.*`                                                        | videochannel tuning                                       |
| `vp8.*`                                                          | vp8channel tuning                                         |
| `sei.fps` / `.batch_size` / `.fragment_size` / `.ack_timeout_ms` | seichannel tuning                                         |
| `gen.amount`                                                     | gen mode: number of rooms to create                       |
| `data`                                                           | path to data directory                                    |
| `debug`                                                          | verbose logging                                           |
| `ffmpeg`                                                         | path to ffmpeg binary                                     |
