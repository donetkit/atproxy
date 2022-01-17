# for exit
socks_addr(tailscale_addr() + ":10000")
http_addr(tailscale_addr() + ":10086")

# for local
#socks_addr("0.0.0.0:10000")
#http_addr("0.0.0.0:10086")
#upstream("100.118.120.94:10000")
#upstream("100.111.147.87:10000")
#upstream("100.73.249.57:10000")
#upstream("100.68.74.6:10000")
#no_direct("github")

