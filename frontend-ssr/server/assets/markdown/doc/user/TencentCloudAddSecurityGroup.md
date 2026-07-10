# 腾讯云自动化添加安全组

-   [前置项](#前置项)
-   [1\. 创建参数模板，在安全组中引用](#_1-创建参数模板-在安全组中引用)
-   [2\. 准备代码](#_2-准备代码)
-   [3\. 编译](#_3-编译)
-   [reference](#reference)
-   [IPv6工具箱 小程序 和 iOS App](#ipv6工具箱-小程序-和-ios-app)
    -   [小程序](#小程序)
    -   [苹果 iOS App](#苹果-ios-app)

一般登陆云服务器都限制来源 IP，添加白名单较为繁琐，通过接口可以快速添加。


```bash
# ./addWhiteIP

{"Response":{"RequestId":"c89ddf9df-738c-4f2a-9f02-dcd3cfe8c852"}}
Outer IP : 106.224.145.147
```


## 前置项

在腾讯云用户管理页面创建一个用于操作私有网络（VPC）的权限

-   访问方式：编程访问
-   用户权限：`QcloudVPCFullAccess`

![-w1796](/doc/16129597411766.jpg)

## 1\. 创建参数模板，在安全组中引用

直接用 API 频繁修改安全组存在风险，我们可以将[参数模板](https://console.cloud.tencent.com/vpc/tpl)作为安全组中的一行。

![w1758](/doc/16129604712653.jpg)

在[安全组](https://console.cloud.tencent.com/vpc/securitygroup)中引用即可。

![w1733](/doc/16129602811940.jpg)

## 2 准备代码

请参照 [腾讯云的 API explorer](https://console.cloud.tencent.com/api/explorer?Product=vpc&Version=2017-03-12&Action=ModifyAddressTemplateAttribute&SignVersion=) 生成对应的代码。


```go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// VPC 中添加 IP组的方式管理安全组

func main() {

	credential := common.NewCredential(
		"<SecretId>",  // 请替换为你的腾讯云 SecretId
		"<SecretKey>", // 请替换为你的腾讯云 SecretKey
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "vpc.tencentcloudapi.com"
	client, _ := vpc.NewClient(credential, "<Region>", cpf) // 请替换为可用区ID，例如 ap-guangzhou

	request := vpc.NewModifyAddressTemplateAttributeRequest()

    // 获取IPv4地址：https://4.wsmdn.dpdns.org
	// 获取IPv6地址：https://6.wsmdn.dpdns.org
	responseClient, errClient := http.Get("https://4.wsmdn.dpdns.org") // 获取外网 IP
	if errClient != nil {
		fmt.Printf("获取外网 IP 失败，请检查网络\n")
		panic(errClient)
	}
	// 程序在使用完 response 后必须关闭 response 的主体。
	defer responseClient.Body.Close()

	body, _ := ioutil.ReadAll(responseClient.Body)

	clientIP := string(body)
	params := fmt.Sprintf("{\"AddressTemplateId\":\"<AddressTemplateId>\",\"Addresses\":[\"%s\"]}", clientIP) // 请将 AddressTemplateId 替换为参数模板 ID
	err := request.FromJsonString(params)
	if err != nil {
		panic(err)
	}
	response, err := client.ModifyAddressTemplateAttribute(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return
	}
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", response.ToJsonString())
	fmt.Printf("\nOuter IP : %s\n", clientIP)

}
```


## 3 编译


```bash
# go build addWhiteIP.go
# ./addWhiteIP
{"Response":{"RequestId":"ec9d18dc-cfcf-4f0b-a0c0-02a9c212dcxx"}}
Outer IP : xx.xx.xx.xx
```


## reference

-   [1] TencentCloud. [腾讯云安全组](https://console.cloud.tencent.com/api/explorer?Product=vpc&Version=2017-03-12&Action=ModifyAddressTemplateAttribute&SignVersion=)

