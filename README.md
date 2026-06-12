# zte-cpe-go

[English](#english) | [中文](./README_zh.md)

A Go CLI tool and library for interacting with ZTE CPE routers, such as the ZTE MF289F and ZTE G5TS.

This project is a **Go port** of [zte-cpe-rs](https://github.com/1zun4/zte-cpe-rs) — a Rust library by [Izuna](https://github.com/1zun4). All credit for the original API reverse engineering goes to the upstream project.

## Supported Devices

- ZTE G5TS (5G CPE)
- ZTE MF289F (GigaCube, LTE CPE)

## Features

| Feature | MF289F | G5TS |
| --- | --- | --- |
| Device reboot | Yes | Yes |
| Get status info | Yes | Yes |
| Get device info | No | Yes |
| Get network/signal information | No | Yes |
| Get SIM card info | No | Yes |
| Connect and disconnect network | Yes | Yes |
| Set connection mode | Yes | Yes |
| Set bearer preference | Yes | Yes |
| Set LTE band lock | Yes | No |
| Set DNS mode | Yes | No |
| Configure UPnP | Yes | Yes |
| Configure DMZ | Yes | Yes |
| Get APN profiles | No | Yes |
| Modify an APN profile | No | Yes |
| Get DHCP settings | No | Yes |
| Set DHCP settings | No | Yes |
| Get MTU/MSS settings | No | Yes |
| Set MTU/MSS settings | No | Yes |
| Get SMS settings | No | Yes |
| Get connected devices | No | Yes |
| Prometheus exporter | Yes | Yes |

## Installation

```sh
go install github.com/chennest/zte-cpe-go@latest
```

Or clone and build:

```sh
git clone https://github.com/chennest/zte-cpe-go.git
cd zte-cpe-go
go build -o zte-cpe .
```

## Usage

### CLI Commands

```sh
zte-cpe status -t g5ts -u http://192.168.0.1 -p YOURPASSWORD
zte-cpe version -t g5ts -u http://192.168.0.1 -p YOURPASSWORD
zte-cpe network-info -t g5ts -u http://192.168.0.1 -p YOURPASSWORD --pretty
zte-cpe sim-info -t g5ts -u http://192.168.0.1 -p YOURPASSWORD --pretty
zte-cpe device-info -t g5ts -u http://192.168.0.1 -p YOURPASSWORD --pretty
zte-cpe connected-devices -t g5ts -u http://192.168.0.1 -p YOURPASSWORD --pretty
zte-cpe get-apn -t g5ts -u http://192.168.0.1 -p YOURPASSWORD
zte-cpe get-dhcp -t g5ts -u http://192.168.0.1 -p YOURPASSWORD
zte-cpe get-mtu -t g5ts -u http://192.168.0.1 -p YOURPASSWORD
zte-cpe get-sms-settings -t g5ts -u http://192.168.0.1 -p YOURPASSWORD
```

### Prometheus Exporter

Start a metrics server that exposes router data to Prometheus:

```sh
zte-cpe serve -t g5ts -u http://192.168.0.1 -p YOURPASSWORD --listen :9101 --interval 30
```

All flags can also be set via environment variables:

| Flag | Env Variable | Default |
| --- | --- | --- |
| `--type` | `ZTE_TYPE` | *(required)* |
| `--url` | `ZTE_URL` | *(required)* |
| `--password` | `ZTE_PASSWORD` | *(required)* |
| `--listen` | `ZTE_LISTEN` | `:9101` |
| `--interval` | `ZTE_INTERVAL` | `30` |

**Example metrics output:**

```
zte_cpe_signal_rsrp_dbm{model="g5ts",network_type="SA"} -82
zte_cpe_signal_rsrq_db{model="g5ts",network_type="SA"} -11
zte_cpe_signal_snr_db{model="g5ts",network_type="SA"} 15.5
zte_cpe_signal_bar{model="g5ts"} 5
zte_cpe_network_connected{model="g5ts"} 1
zte_cpe_connected_devices{model="g5ts"} 0
zte_cpe_device_info{firmware="BD_SEECOMCNG5TSV1.0.0B05",hardware_version="G5TSHW_1.0.0",imei="869338073140877",mac_address="5C:7D:AE:AF:3B:67",model="g5ts",network_type="SA",operator="UNICOM"} 1
```

### Docker

Pre-built images are available on GHCR:

```sh
docker pull ghcr.io/chennest/zte-cpe-go:latest
```

Or build from source:

```sh
docker build -t zte-cpe-go .
```

Run:

```sh
docker run -d --name zte-cpe-exporter \
  --network host \
  -e ZTE_TYPE=g5ts \
  -e ZTE_URL=http://192.168.0.1 \
  -e ZTE_PASSWORD=YOURPASSWORD \
  -p 9101:9101 \
  ghcr.io/chennest/zte-cpe-go:latest
```

**Prometheus scrape config:**

```yaml
scrape_configs:
  - job_name: 'zte-cpe'
    static_configs:
      - targets: ['localhost:9101']
```

### As a Library

```go
package main

import (
    "context"
    "fmt"

    "github.com/chennest/zte-cpe-go/pkg/g5ts"
    "github.com/chennest/zte-cpe-go/pkg/router"
)

func main() {
    client, err := g5ts.New("http://192.168.0.1")
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    if err := client.Login(ctx, "YOURPASSWORD"); err != nil {
        panic(err)
    }
    defer client.Logout(ctx)

    status, err := client.GetStatus(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println(string(status))
}
```

## Acknowledgements

- [zte-cpe-rs](https://github.com/1zun4/zte-cpe-rs) — Original Rust implementation this project is ported from
- [ZTE-MC-Home-assistant](https://github.com/Kajkac/ZTE-MC-Home-assistant)
- [zte-cpe](https://github.com/SpeckyYT/zte-cpe)
- [zte-v3.0b.min.txt](https://miononno.it/files/zte-v3.0b.min.txt)

## License

This project is licensed under the GNU General Public License v3.0.
