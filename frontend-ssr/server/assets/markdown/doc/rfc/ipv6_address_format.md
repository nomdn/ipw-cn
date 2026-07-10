# IPv6 地址标识方法

我们以 IP查询(ipw.wsmdn.top) 的 IPv6 地址为例来介绍三种 IPv6 地址表示方法：首选格式、压缩格式、内嵌 IPv4 地址的 IPv6 地址格式。


```sh
~$ ping6 ipw.wsmdn.top
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=0 hlim=51 time=16.168 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=1 hlim=51 time=15.788 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=2 hlim=51 time=10.048 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=3 hlim=51 time=16.051 ms
^C
--- ipw.wsmdn.top ping6 statistics ---
4 packets transmitted, 4 packets received, 0.0% packet loss
round-trip min/avg/max/std-dev = 10.048/14.514/16.168/2.582 ms
```


## 首选格式

最严谨的表示方法，但空值很多。


```sh
~$ ping6 2402:4e00:0040:0040:0000:0000:2:3b6
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=0 hlim=51 time=9.898 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=1 hlim=51 time=16.233 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=2 hlim=51 time=23.645 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=3 hlim=51 time=16.250 ms
--- 2402:4e00:0040:0040:0000:0000:2:3b6 ping6 statistics ---
4 packets transmitted, 4 packets received, 0.0% packet loss
round-trip min/avg/max/std-dev = 9.898/16.506/23.645/4.868 ms
```


## 压缩格式

-   将 将前置 0 去掉 0040:0040 -> 40:40

-   连续多个 0 以 :: 替代，最多出现 1 次 :: 0000:0000 -> ::



```sh
~$ ping6 2402:4e00:40:40::2:3b6
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=0 hlim=51 time=16.182 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=1 hlim=51 time=16.252 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=2 hlim=51 time=15.984 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=3 hlim=51 time=15.740 ms
^C
--- 2402:4e00:40:40::2:3b6 ping6 statistics ---
4 packets transmitted, 4 packets received, 0.0% packet loss
round-trip min/avg/max/std-dev = 15.740/16.040/16.252/0.199 ms
```


注：在 ping6 程序中可以看到最后都转换为 压缩格式。

## 内嵌 IPv4 地址的 IPv6 地址格式

将 IPv6 地址最后两个 16 位从 16 进制转成 10 进制，用点分十进制来表示。

:2:3b6 -> 0.2.3.182


```sh
~$ ping6 2402:4e00:40:40::0.2.3.182
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=0 hlim=51 time=9.721 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=1 hlim=51 time=9.742 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=2 hlim=51 time=9.637 ms
16 bytes from 2402:4e00:40:40::2:3b6, icmp_seq=3 hlim=51 time=16.181 ms
^C
--- 2402:4e00:40:40::0.2.3.182 ping6 statistics ---
4 packets transmitted, 4 packets received, 0.0% packet loss
round-trip min/avg/max/std-dev = 9.637/11.320/16.181/2.807 ms
```


## 其他

### 本地回环地址


```sh
~$ ping6 0:0:0:0:0:0:0:1
PING6(56=40+8+8 bytes) ::1 --> ::1
16 bytes from ::1, icmp_seq=0 hlim=64 time=0.057 ms
16 bytes from ::1, icmp_seq=1 hlim=64 time=0.057 ms
16 bytes from ::1, icmp_seq=2 hlim=64 time=0.081 ms
16 bytes from ::1, icmp_seq=3 hlim=64 time=0.088 ms
^C
--- 0:0:0:0:0:0:0:1 ping6 statistics ---
4 packets transmitted, 4 packets received, 0.0% packet loss
round-trip min/avg/max/std-dev = 0.057/0.071/0.088/0.014 ms
```


