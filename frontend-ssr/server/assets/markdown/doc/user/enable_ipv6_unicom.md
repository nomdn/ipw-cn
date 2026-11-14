# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 联通SK-D742-C光猫管理员密码获取/宽带密码获取/改桥接/IPv6设置

> 本文转载自「JZX 轻语」，作者陈建彰。原文采用 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/) 许可协议，本站按同一协议署名转载，出处与协议全文见文末。
>
> 提醒：以下操作仅适用于你自己名下的光猫设备；改桥接前请把原有连接配置、宽带账号密码记录下来，改错可能导致上不了网。

## 前言

最近家里网络波动有点大，感觉是光猫的性能不够好，遂尝试把光猫从路由模式改成桥接模式，并顺便修复至今存在的 IPv6 地址能获取的、但一直无法访问 IPv6 网站的问题。之前装网络的时候比较急，并没有叫师傅调整光猫的模式，且对宽带密码和光猫的管理员账户密码一无所知，恰逢过年也懒得打扰客服，于是用了一个下午的时间研究破解光猫并从中提取出这些信息，并将光猫改为桥接模式，使用路由器进行拨号。

家里的设备和操作环境如下，如果你的光猫或路由器 IP 和我的不一致，下文与这些设备有关的网址都需要根据自己的情况进行调整：

- 宽带： 广州联通 1000M 光纤
- 光猫：创维 SK-D742-C，IP：192.168.1.1
- 路由器：红米 AX6S，IP：192.168.31.1
- 路由器的 WAN 口通过网线接在光猫的 LAN1（千兆口）上
- 电脑操作系统：Ubuntu 22.04 (其实很多命令在 Windows 也是差不多的，因为常用 Ubuntu，所以对光猫的设置都在 Ubuntu 上进行)

## Step 1：获取管理员密码

首先在浏览器打开 <http://192.168.1.1/hidden_version_switch.html>，然后勾选 Telnet Enable 选项，此时网页会闪烁刷新一下。页面刷新后 Telnet Enable 选项可能还是处于未选择状态，这时候不用理会，继续下一步。

![勾选 Telnet Enable 选项](/docimg/unicom_modem_01.jpg)

## Step 2: 尝试 Telnet 登录光猫

进入终端（or 命令提示符），输入 `telnet 192.168.1.1`，观察是否出现「Connected to 192.168.1.1」，如有，则说明 Telnet 开启成功。此时关闭终端即可，不需要登录。

![尝试使用 telnet 登录光猫](/docimg/unicom_modem_02.jpg)

## Step 3：通过 FTP 获取管理员密码

在文件管理器中输入 <ftp://192.168.1.1>，帐号和密码都是 useradmin。登录成功后，进入 `/tmp` 目录，找到 telnet_su_passwd 文件，如下图所示：

![通过 FTP 获取管理员密码](/docimg/unicom_modem_03.jpg)

打开文件，文件内容为光猫 WiFi 密码拼接管理员密码的结果，如下图所示，我的光猫 WiFi 密码是 truumf3（默认密码可以在光猫下方标签找到），而后面的字符串即为光猫管理员密码！

![telnet_su_passwd 文件内容](/docimg/unicom_modem_04.jpg)

## Step 4：继续通过 FTP 获取宽带密码

一般来说，宽带密码其实可以找客服就可以拿到了，但大过年的，客服不一定能及时回复，所以还是自己继续探索 `/tmp` 文件夹找找看吧。在 `/tmp` 文件夹中，发现了一个名为 ppp 的文件夹，这正是宽带拨号所产生的配置临时文件所处的地方！进入 `/tmp/ppp` 文件夹，找到 options.nbif1 文件（或者诸如 options.nbifx 的形式，"x" 可能是阿拉伯数字）。

![/tmp/ppp 文件夹](/docimg/unicom_modem_05.jpg)

打开文件，即可看到 PPP 拨号的配置信息。如下图所示，user 对应的是宽带帐号（不放心的话可以和联通官网上查到的帐号比对以下），password 即为宽带密码，然后也可以顺便记录下 mru、mtu 的信息（可选，应该不影响网络），尽量保持和光猫拨号的配置一致。

![options.nbif1 文件内容](/docimg/unicom_modem_06.jpg)

## Step 5：以管理员身份登录光猫

进入网页 <http://192.168.1.1/CU.html>，输入帐号 CUAdmin 和先前获取到的管理员密码，即可进入光猫管理页面。

![登录光猫管理页面](/docimg/unicom_modem_07.jpg)

## Step 6：光猫改桥接模式

此时，可以以管理员的身份调整光猫的配置。进入基本配置 -> 上行线路配置，在连接名称中找到带有 X_INTERNET_R_VID_X 样式（其中 X 指代阿拉伯数字，因设备而异，第一个 X 表示配置序号，第二个 X 是 VLAN ID）的连接配置，然后记录下该配置所绑定的端口（一般为 LAN1 千兆口，如果该配置不是 LAN1 的话务必多次确认看看有没有找到正确的配置了）、VLAN 模式和 VLAN ID，最后删除该连接配置。

![记录路由模式相应的配置](/docimg/unicom_modem_08.jpg)

新建一个 WAN 连接（上行线路配置页面在连接名称上选择新建 WAN 连接），将 IP 协议版本调整为双栈的 IPv4/v6，模式调整为桥模式，端口绑定为 LAN1（根据自己的实际情况，一般都是 LAN1，即路由器接在光猫上的端口），勾选 DHCP 服务使能，业务模式保持为 INTERNET，VLAN 模式和 VLAN ID 设置为刚刚所记录的配置值，后面的配置保持不变。

![桥模式配置](/docimg/unicom_modem_09.jpg)

## Step 7：在路由器配置中设置 PPPoE 拨号

(该步骤根据路由器型号而异)

登录路由器配置，将连接模式改为 PPPoE 拨号，输入宽带帐号和密码，以及 MTU 一些信息，即可完成光猫改桥接模式、路由器拨号上网的操作~

![路由器配置](/docimg/unicom_modem_10.jpg)

## Step 8：结果

此时可以发现，路由器已经可以成功拨号并获取到相应的 IPv4 和 IPv6 地址。通过 <http://www.test-ipv6.com/> 可以发现此时已经通过了所有的 IPv6 测试。

![路由器拨号状态，已获取 IPv4 与 IPv6 地址](/docimg/unicom_modem_11.jpg)
![IPv6 连接测试结果，全部通过](/docimg/unicom_modem_12.jpg)

---

**版权声明**：本博客所有文章除特别声明外，均采用 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/) 许可协议。转载请注明出处。

| 项 | 内容 |
| --- | --- |
| 原文标题 | 联通SK-D742-C光猫管理员密码获取/宽带密码获取/改桥接/IPv6设置 |
| 作者 | 陈建彰（JZX 轻语） |
| 原文链接 | <https://jeza-chen.com/2023/01/23/Playing-with-Modem/> |
| 原文发布时间 | 2023-01-23 |
| 原文许可 | [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/)（署名 — 非商业性使用 — 相同方式共享） |
| 本站改动 | 图片转存到本站并压缩、补写图注，开头增加一行转载说明与安全提醒；正文文字未删改。本站为个人非商业站点，本页同样以 CC BY-NC-SA 4.0 协议共享。 |
