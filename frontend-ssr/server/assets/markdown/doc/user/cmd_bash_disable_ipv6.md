# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> macOS和 Window10 命令行禁用/启用IPv6本地网络

图形化禁用 IPv6 网络比较繁琐，以下是 macOS 和 Window 10 命令行下禁用/启用 IPv6 本地网络的方法。

· macOS 禁用/启用IPv6本地网络
· Window10 禁用或启用 IPv6 本地网络
· 附录
· macOS 和 Window10 禁用/启用 IPv4 本地网络
    · macOS 禁用/启用IPv4本地网络
    · Window10 禁用/启用IPv4本地网络
## macOS 禁用/启用IPv6本地网络
- 禁用 IPv6 本地网络
```sh
# 验证是否可以访问 IPv6 网络
$ curl 6.ipw.cn
2408:824c:200::2b8b:336f:cc9c

# 找到网络设备的名字，这里用的是 Wi-Fi （因为 macbook 用的是无线网络访问，即 Wi-Fi）
$ networksetup -listallnetworkservices
An asterisk (*) denotes that a network service is disabled.
Wi-Fi
Bluetooth PAN

# 禁用 Wi-Fi 设备的 IPv6 本地网络
$ networksetup -setv6off Wi-Fi

# 请求 IPv4/IPv6 双栈网络，返回 IPv4 地址，证明 IPv4 访问优先。 
$ curl test.ipw.cn
159.75.190.197

# 访问 IPv6 网络失败 
$ curl 6.ipw.cn
curl: (6) Could not resolve host: 6.ipw.cn
```
- 启用 IPv6 本地网络
```sh
# 启用 Wi-Fi 设备的本地 IPv6 网络
$ networksetup -setv6automatic Wi-Fi

# 验证本地 IPv6 网络访问是否成功
$ curl test.ipw.cn
2408:824c:200::2b8b:336f:cc9c
```
## Window10 禁用或启用 IPv6 本地网络
以管理员身份打开 PowerShell，执行以下命令。

- 禁用 IPv6 本地网络
```sh
# 查看当前 IPv6 网络设备
PS C:\WINDOWS\system32>  Get-NetAdapterBinding -ComponentID ms_tcpip6

Name                           DisplayName                                        ComponentID          Enabled
----                           -----------                                        -----------          -------
WLAN                           Internet 协议版本 6 (TCP/IPv6)                     ms_tcpip6            True
以太网                         Internet 协议版本 6 (TCP/IPv6)                     ms_tcpip6            True

# 验证本地 IPv6 网络是否访问成功
PS C:\WINDOWS\system32> curl 6.ipw.cn                                                                                                                                                                                                                                                                                                                                   
Content           : 2408:824c:200::2b8b:336f:cc9c

# 禁用 WLAN 设备的本地 IPv6 网络（因为笔记本用的是 无线，即  WLAN）
PS C:\WINDOWS\system32> Disable-NetAdapterBinding -Name "WLAN" -ComponentID ms_tcpip6

# 查看当前 IPv6 网络使用情况，可以看到 WLAN 的已经禁用掉
PS C:\WINDOWS\system32>  Get-NetAdapterBinding -ComponentID ms_tcpip6

Name                           DisplayName                                        ComponentID          Enabled
----                           -----------                                        -----------          -------
WLAN                           Internet 协议版本 6 (TCP/IPv6)                     ms_tcpip6            False
以太网                         Internet 协议版本 6 (TCP/IPv6)                     ms_tcpip6            True

# 无法访问 IPv6 网络
PS C:\WINDOWS\system32> curl 6.ipw.cn
curl : 无法连接到远程服务器

PS C:\WINDOWS\system32> curl 4.ipw.cn
Content           : 159.75.190.197
```
- 启用 IPv6 本地网络
```sh
# 启用 WLAN 设备的 IPv6 网络
PS C:\WINDOWS\system32> Enable-NetAdapterBinding -Name "WLAN" -ComponentID ms_tcpip6

# 验证 IPv6 网络是否访问成功
PS C:\WINDOWS\system32> curl 6.ipw.cn

Content           : 2408:824c:200::2b8b:336f:cc9c
```
## 附录
### macOS 和 Window10 禁用/启用 IPv4 本地网络
禁用 IPv4 网络，就可以只有 IPv6 网络访问了。

### macOS 禁用/启用IPv4本地网络
禁用 IPv4 本地网络
```sh
# 禁用 Wi-Fi 设备的本地 IPv4 网络
$ networksetup -setv4off Wi-Fi

# 访问 IPv4 网络失败
$ curl 4.ipw.cn
curl: (6) Could not resolve host: 4.ipw.cn
```

- 启用 IPv4 本地网络
```sh
$ networksetup -setdhcp Wi-Fi

$ curl 4.ipw.cn
159.75.190.197
```
### Window10 禁用/启用IPv4本地网络
以管理员身份打开 PowerShell，执行以下命令。

- 禁用 IPv4 本地网络
```sh
PS C:\WINDOWS\system32> Disable-NetAdapterBinding -Name "WLAN" -ComponentID ms_tcpip

PS C:\WINDOWS\system32> curl 4.ipw.cn
curl : 无法连接到远程服务器
```
- 启用 IPv4 本地网络
```sh
PS C:\WINDOWS\system32> Enable-NetAdapterBinding -Name "WLAN" -ComponentID ms_tcpip

PS C:\WINDOWS\system32> curl 4.ipw.cn
159.75.190.197
```