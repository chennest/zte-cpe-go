# zte-cpe-go

[中文](#中文) | [English](./README.md)

## 中文

一个用 Go 编写的 ZTE CPE 路由器管理工具和库，支持 ZTE MF289F（GigaCube）和 ZTE G5TS（5G CPE）设备。

本项目是 [zte-cpe-rs](https://github.com/1zun4/zte-cpe-rs) 的 **Go 语言移植版**，原项目由 [Izuna](https://github.com/1zun4) 使用 Rust 编写。所有路由器 API 的逆向工程工作归功于原作者。

### 快速开始

```sh
# 安装
go install github.com/chennest/zte-cpe-go@latest

# 查看路由器状态
zte-cpe status -t g5ts -u http://192.168.0.1 -p 你的密码

# 查看网络信号信息
zte-cpe network-info -t g5ts -u http://192.168.0.1 -p 你的密码 --pretty

# 查看 SIM 卡信息
zte-cpe sim-info -t g5ts -u http://192.168.0.1 -p 你的密码 --pretty

# 查看已连接设备
zte-cpe connected-devices -t g5ts -u http://192.168.0.1 -p 你的密码 --pretty
```

### Prometheus 监控

启动 metrics 服务，将路由器数据暴露给 Prometheus：

```sh
zte-cpe serve -t g5ts -u http://192.168.0.1 -p 你的密码 --listen :9101 --interval 30
```

所有参数均可通过环境变量设置：

| 参数 | 环境变量 | 默认值 |
| --- | --- | --- |
| `--type` | `ZTE_TYPE` | *（必填）* |
| `--url` | `ZTE_URL` | *（必填）* |
| `--password` | `ZTE_PASSWORD` | *（必填）* |
| `--listen` | `ZTE_LISTEN` | `:9101` |
| `--interval` | `ZTE_INTERVAL` | `30` |

**指标示例：**

```
zte_cpe_signal_rsrp_dbm{model="g5ts",network_type="SA"} -82
zte_cpe_signal_rsrq_db{model="g5ts",network_type="SA"} -11
zte_cpe_signal_snr_db{model="g5ts",network_type="SA"} 15.5
zte_cpe_signal_bar{model="g5ts"} 5
zte_cpe_network_connected{model="g5ts"} 1
zte_cpe_connected_devices{model="g5ts"} 0
zte_cpe_device_info{firmware="BD_SEECOMCNG5TSV1.0.0B05",hardware_version="G5TSHW_1.0.0",imei="869338073140877",mac_address="5C:7D:AE:AF:3B:67",model="g5ts",network_type="SA",operator="UNICOM"} 1
```

### Docker 部署

```sh
docker build -t zte-cpe-go .

docker run -d --name zte-cpe-exporter \
  --network host \
  -e ZTE_TYPE=g5ts \
  -e ZTE_URL=http://192.168.0.1 \
  -e ZTE_PASSWORD=你的密码 \
  -p 9101:9101 \
  zte-cpe-go
```

**Prometheus 采集配置：**

```yaml
scrape_configs:
  - job_name: 'zte-cpe'
    static_configs:
      - targets: ['localhost:9101']
```

### 支持的功能

| 功能 | MF289F | G5TS |
| --- | --- | --- |
| 重启路由器 | ✅ | ✅ |
| 获取状态信息 | ✅ | ✅ |
| 获取设备信息 | ❌ | ✅ |
| 获取网络/信号信息 | ❌ | ✅ |
| 获取 SIM 卡信息 | ❌ | ✅ |
| 连接/断开网络 | ✅ | ✅ |
| 设置连接模式 | ✅ | ✅ |
| 设置网络制式 | ✅ | ✅ |
| 锁定 LTE 频段 | ✅ | ❌ |
| 设置 DNS | ✅ | ❌ |
| 配置 UPnP | ✅ | ✅ |
| 配置 DMZ | ✅ | ✅ |
| 获取 APN 配置 | ❌ | ✅ |
| 修改 APN 配置 | ❌ | ✅ |
| 获取 DHCP 设置 | ❌ | ✅ |
| 设置 DHCP | ❌ | ✅ |
| 获取 MTU/MSS 设置 | ❌ | ✅ |
| 设置 MTU/MSS | ❌ | ✅ |
| 获取短信设置 | ❌ | ✅ |
| 获取已连接设备 | ❌ | ✅ |
| Prometheus 监控指标 | ✅ | ✅ |

### 作为库使用

```go
package main

import (
    "context"
    "fmt"

    "github.com/chennest/zte-cpe-go/pkg/g5ts"
)

func main() {
    client, err := g5ts.New("http://192.168.0.1")
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    if err := client.Login(ctx, "你的密码"); err != nil {
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

### 致谢

- [zte-cpe-rs](https://github.com/1zun4/zte-cpe-rs) — 本项目移植自该 Rust 原版
- [ZTE-MC-Home-assistant](https://github.com/Kajkac/ZTE-MC-Home-assistant)
- [zte-cpe](https://github.com/SpeckyYT/zte-cpe)
- [zte-v3.0b.min.txt](https://miononno.it/files/zte-v3.0b.min.txt)

### 许可证

本项目基于 GNU General Public License v3.0 许可证发布。
