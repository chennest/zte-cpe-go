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
