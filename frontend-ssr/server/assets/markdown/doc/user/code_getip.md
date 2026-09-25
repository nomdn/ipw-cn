# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> Python/Golang获取 IPv4 和 IPv6 地址

> 本站是个人站点，用于作者实战学习前端和后台知识，请勿用于商业用途，仅供个人测试学习之用，请遵守中国法律法规

通过 Python/Golang 获取公网 IPv4 和 IPv6 地址，还可以返回是 IPv4 还是 IPv6 访问优先。

## 1. Python/Golang获取 IPv4 地址

### 1.1 Python 获取本机公网IPv4地址

- 输入示例

```python
#!/usr/bin/python3

import requests

# 获取 IPv4 地址接口； https://4.wsmdn.top
r = requests.get('https://4.wsmdn.top')

clientIP = r.text

print(clientIP)
```

- 输出示例

```
106.224.145.147
```

### 1.2 Golang 获取本机公网IPv4 地址

- 输入示例

```go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {

    // 获取 IPv4 地址接口； https://4.wsmdn.top
	responseClient, errClient := http.Get("https://4.wsmdn.top")

	if errClient != nil {
		fmt.Printf("获取外网 IP 失败，请检查网络\n")
		panic(errClient)
	}
	defer responseClient.Body.Close()

	// 获取 http response 的 body
	body, _ := ioutil.ReadAll(responseClient.Body)
	clientIP := string(body)

	print(clientIP)

}
```

- 输出示例

```
106.224.145.147
```

## 2. Python/Golang获取 IPv6 地址

### 2.1 Python 获取本机公网IPv6地址

- 输入示例

```python
#!/usr/bin/python3

import requests

r = requests.get('https://6.wsmdn.top')

clientIP = r.text

print(clientIP)
```

- 输出示例

```
2408:824c:200::2b8b:336f:cc9c
```

### 2.2 Golang 获取本机公网IPv6地址

- 输入示例

```go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {

	responseClient, errClient := http.Get("https://6.wsmdn.top")

	if errClient != nil {
		fmt.Printf("获取外网 IP 失败，请检查网络\n")
		panic(errClient)
	}
	defer responseClient.Body.Close()

	// 获取 http response 的 body
	body, _ := ioutil.ReadAll(responseClient.Body)
	clientIP := string(body)

	print(clientIP)

}
```

- 输出示例

```
2408:824c:200::2b8b:336f:cc9c
```

## 3. 测试网络是IPv4还是IPv6访问优先

### 3.1 Python 测试网络是IPv4还是IPv6访问优先

- 输入示例

```python
#!/usr/bin/python3

import requests

## HTTP GET，返回的是纯文本地址
r = requests.get('https://test.wsmdn.top')

IP = r.text.strip()

## 返回的地址含冒号即 IPv6，否则为 IPv4
IPVersion = 'IPv6' if ':' in IP else 'IPv4'

## 打印
print('IP:',IP)
print('IPVersion:',IPVersion)
```

- 输出示例

```bash
## 返回示例1（IPv4 访问优先）
IP: 106.224.145.147
IPVersion: IPv4

## 返回示例2（IPv6 访问优先）
IP: 2408:824c:200::2b8b:336f:cc9c
IPVersion: IPv6
```

### 3.2 Golang 测试网络是IPv4还是IPv6访问优先

- 输入示例

```go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func main() {

	responseClient, errClient := http.Get("https://test.wsmdn.top") // 获取外网 IP
	if errClient != nil {
		fmt.Printf("获取外网 IP 失败，请检查网络\n")
		panic(errClient)
	}
	// 程序在使用完 response 后必须关闭 response 的主体。
	defer responseClient.Body.Close()

	body, _ := ioutil.ReadAll(responseClient.Body)

	IP := strings.TrimSpace(string(body))

	// 返回的地址含冒号即 IPv6，否则为 IPv4
	IPVersion := "IPv4"
	if strings.Contains(IP, ":") {
		IPVersion = "IPv6"
	}

	fmt.Printf("IP: %s\n", IP)
	fmt.Printf("IPVersion: %s\n", IPVersion)
}
```

- 输出示例

```bash
## 返回示例1（IPv4 访问优先）
IP: 106.224.145.147
IPVersion: IPv4

## 返回示例2（IPv6 访问优先）
IP: 2408:824c:200::2b8b:336f:cc9c
IPVersion: IPv6
```
