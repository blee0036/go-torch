# go-torch

![build](https://github.com/blee0036/go-torch/actions/workflows/build.yml/badge.svg?branch=master)

Torch backend rewrote on Golang with ARM support

## Usage

- Ping TCP port
  - `/ping/:host/:port`
  - onSuccess: `{"status":true,"time":975.9993876666667}`
  - onFailure(3 tries): `{"status":false,"time":0}`
- Resolve given addr
  - `/resolve/:host` using default resolver
  - onSuccess: `{"result":["220.181.38.148","39.156.69.79"],"status":true,"time":34.33183533333334}`
  - onFailure(3 tries): `{"result":null,"status":false,"time":0}`

## Download

### Binary 

[Release](releases)

###  Docker 

`ghcr.io/blee0036/go-torch`

```bash
docker run -d -p 8080:8080 ghcr.io/blee0036/go-torch:latest
```
|key|required|default|
|---|---|---|
|PORT|false|8080|
|GIN_MODE|false|production|
|CORS_ALLOW_ORIGINS|false|disabled; comma-separated origins, for example `https://example.com`|
|RATE_LIMIT_RPM|false|100; set to `-1` to disable the global limit|


When the global limit is exceeded, the service returns HTTP `200` with
`{"data":"rate limit"}`.

The `/ping` and `/resolve` endpoints intentionally connect to or resolve the
requested host. Only public IPv4 and IPv6 addresses are allowed; private,
loopback, link-local, shared, reserved, documentation, and multicast ranges
are rejected. Do not expose this service directly to untrusted clients
without authentication and rate limiting.
