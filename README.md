# Marco

A service that aggregates Vitess statistics for Prometheus

### Build Setup
``` bash
# fetch dependencies using govendor
$ govendor sync
# build golang source code
$ cd go/marco && go build -o marco .
```

### Usage
``` bash
$ ./marco --port [http server port] --vtgate [vtgate address] --etcd [global etcd address] --cells [cell names to monitor]
```

### Exported stats

#### Vttablet
exported vttablet stats are available at /vttablet and contian following statistics
 * total qps of tablet
### Vtgate
 * Vtgate total time
 * Vtgate total count
 * count and time for all tablets in order to compute tablet latency
