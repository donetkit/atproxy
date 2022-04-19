pool_capacity(512)
pool_buffer_size(4 * 1024)

# for exit
server(
    tailscale_addr() + ":10000",
    tailscale_addr() + ":10086",
)

