# atproxy
Socks5 proxy server with auto upstream selection

Atproxy selects upstream by real-time connection reachability and latency,
eliminating needs for whitelist/blacklist rules.

## installation
```
go install github.com/reusee/atproxy/atproxy@master
```

## select process
1. For each client connection, open connections to all upstream servers
2. Also open a direct connection to the target address
3. Send client data to all opened connections
4. Wait for the first connection that has inbound data, which is usually the optimal one
5. Continue exchanging data with the selected connection
6. Close other non-optimal connections

