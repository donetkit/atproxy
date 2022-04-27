pool_capacity(512)
pool_buffer_size(4 * 1024)

# for exit
server_spec(
    Socks = tailscale_addr() + ":10000",
    HTTP = tailscale_addr() + ":10086",
)

