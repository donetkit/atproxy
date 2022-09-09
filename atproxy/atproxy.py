# for exit
#server(
#    tailscale_addr() + ":10000",
#    tailscale_addr() + ":10086",
#)

# for local
server_spec(
    Socks = "0.0.0.0:10000",
    HTTP = "0.0.0.0:10086",
    Upstreams = {
        "HK1": "100.118.120.94:10000",
        "JP1": "100.111.147.87:10000",
        "US1": "100.73.249.57:10000",
        "US2": "100.68.74.6:10000",
    },
)
server_spec(
    Socks = "0.0.0.0:11000",
    HTTP = "0.0.0.0:11086",
    Upstreams = {
        "HK1": "100.118.120.94:10000",
    },
    NoDirect = True,
)
server_spec(
    Socks = "0.0.0.0:12000",
    HTTP = "0.0.0.0:12086",
    Upstreams = {
        "JP1": "100.111.147.87:10000",
    },
    NoDirect = True,
)
server_spec(
    Socks = "0.0.0.0:13000",
    HTTP = "0.0.0.0:13086",
    Upstreams = {
        "US1": "100.73.249.57:10000",
    },
    NoDirect = True,
)
server_spec(
    Socks = "0.0.0.0:14000",
    HTTP = "0.0.0.0:14086",
    Upstreams = {
        "US2": "100.68.74.6:10000",
    },
    NoDirect = True,
)

no_direct("github")
no_direct("google")
no_direct("gstatic")
no_direct("mypikpak")

no_upstream("163.com")
no_upstream("jd.com")
no_upstream("360buyimg.com")

pool_capacity(512)
pool_buffer_size(4 * 1024)

