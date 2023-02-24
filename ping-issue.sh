# https://github.com/go-ping/ping/issues/191

sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
sudo setcap cap_net_raw,cap_net_admin=eip ./estp

