# atproxy
socks5 proxy server with auto upstream selection

## installation
```
go install github.com/reusee/atproxy/atproxy@master
```

## select process
1. for each client connection, open connections to all upstream servers
2. also open a direct connection to the target address
3. send client data to all opened connections
4. wait for the first connection that has inbound data, which is usually the optimal one
5. continue exchanging data with the selected connection
6. close other non-optimal connections
