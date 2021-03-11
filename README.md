
# bootstrap

1. `dht.Mode(dht.ModeServer)` 或 `dht.Mode(dht.ModeAutoServer)` 来使 `DHT` 环具备 `bootstrap` 性质
2. `p2p`库内部有连接管理器，这样发现节点就可以交叉成网状来了

