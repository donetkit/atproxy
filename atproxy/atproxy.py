# for exit
#server(
#    tailscale_addr() + ":10000",
#    tailscale_addr() + ":10086",
#)

# for local
server(
    "0.0.0.0:10000",
    "0.0.0.0:10086",
    "100.118.120.94:10000",
    "100.111.147.87:10000",
    "100.73.249.57:10000",
    "100.68.74.6:10000",
)
server_spec(
    Socks = "0.0.0.0:20000",
    HTTP = "0.0.0.0:20086",
    Upstreams = ["100.118.120.94:10000"],
    NoDirect = True,
)
server_spec(
    Socks = "0.0.0.0:30000",
    HTTP = "0.0.0.0:30086",
    Upstreams = ["100.111.147.87:10000"],
    NoDirect = True,
)
server_spec(
    Socks = "0.0.0.0:40000",
    HTTP = "0.0.0.0:40086",
    Upstreams = ["100.73.249.57:10000"],
    NoDirect = True,
)
server_spec(
    Socks = "0.0.0.0:50000",
    HTTP = "0.0.0.0:50086",
    Upstreams = ["100.68.74.6:10000"],
    NoDirect = True,
)

no_direct("github")
no_direct("google")

no_upstream("163.com")
no_upstream("jd.com")

pool_capacity(512)
pool_buffer_size(4 * 1024)

