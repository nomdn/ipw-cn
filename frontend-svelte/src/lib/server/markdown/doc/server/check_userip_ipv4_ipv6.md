# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> JS:检查网络是IPv4还是IPv6

> 仅供个人测试学习之用，本站不对稳定性负责。

通过一段 js 代码可以显示用户请求网络是 IPv4 访问优先，还是 IPv6 访问优先。

## 效果

> 如果你没有 IPv6 的网络环境，可以先用手机流量访问该页面。

## 代码

```html
<template>
    <div>
         <a href='https://nipw.cn' target='_blank'>   <b>访客IP:</b> {{IP}}，您的网络 {{IPVersion}} 访问优先</a>
    </div>
</template>

 <script>
  import Axios from "axios";

    export default {
        name: "Home",
        data(){
          return{
            IP: "",
            IPVersion: "",
          }
      },
       headers:
       {
        'X-Requested-With': 'XMLHttpRequest',
        'Access-Control-Allow-Origin': "*"
       },

      methods:{
          getData(){
            var that=this
            // 双栈接口：按本次请求实际走的协议，返回纯文本的 IPv4 或 IPv6 地址
            var api = "https://test.wsmdn.top";

            Axios.get(api)
              .then(function (res) {
                // handle success
                console.log(res.data);
                var ip = String(res.data).trim();
                console.log(ip);
                that.IP = ip
                // 返回的地址含冒号即 IPv6，否则为 IPv4
                that.IPVersion = ip.indexOf(':') > -1 ? "IPv6" : "IPv4"
                console.log(that.IPVersion);
              })
              .catch(function (error) {
                // handle error
                console.log(error);
              })
              .then(function () {
                // always executed
              });
          },
      },
      mounted(){
       this.getData();
      },
    }
</script>
```
