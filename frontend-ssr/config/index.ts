
/*
    前端的一系列配置
*/
const config = {
    siteUrl: "https://ipw.wsmdn.top/",
    // Umami 统计
    umamiSrc: "https://umami.wsmdn.top/zako.js",
    umamiWebsiteId: "69a91329-b110-4cf7-a04a-be4360b1a8d3",
    // Worker IP查询接口
    v4OnlyAPI: "https://4.wsmdn.dpdns.org/",
    v6OnlyAPI: "https://6.wsmdn.dpdns.org/",
    DualStackAPI: "https://test.wsmdn.dpdns.org/",
    // 后端API地址
    apiBaseUrl: 'https://anycast-cloudflare.wsmdn.dpdns.org/',
    IPLocationAPI : "https://cn2-sichuan.api-ipw.wsmdn.top/",
    TCPing:{
        DualStack: [

            {
                label: "中国 上海 Anycast/cloudflare",
                url:"https://anycast-cloudflare.wsmdn.dpdns.org/"
            },
        ],
        IPv4: [
            
            {
                label: "中国 广东 广州 腾讯云",
                url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
            },
            {
                label: "新加坡 腾讯云",
                url :"https://sg-1.api-ipw.wsmdn.top/",
            },
        ],
        IPv6: [
            
            {
                label: "中国 四川 沙渠 电信",
                url:"https://cn2-sichuan.api-ipw.wsmdn.top/",
            }
        ]

    },
    SpeedTest:{
        DualStack: [

            {
                label: "中国 上海 Anycast/cloudflare",
                url:"https://anycast-cloudflare.wsmdn.dpdns.org/"
            },
        ],
        IPv4: [
            
            {
                label: "中国 广东 广州 腾讯云",
                url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
            },
            {
                label: "新加坡 腾讯云",
                url :"https://sg-1.api-ipw.wsmdn.top/",
            },
        ],
        IPv6: [
            
            {
                label: "中国 四川 沙渠 电信",
                url:"https://cn2-sichuan.api-ipw.wsmdn.top/",
            }
        ]
    },
    NSLookup:[
        {
            label: "中国 上海 Anycast/cloudflare",
            url:"https://anycast-cloudflare.wsmdn.dpdns.org/"
        },
        {
            label: "中国 广州 腾讯云",
            url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
        },
        {
            label: "新加坡 腾讯云",
            url :"https://sg-1.api-ipw.wsmdn.top/",
        },
            
        {
            label: "中国 四川 沙渠 电信",
            url:"https://cn2-sichuan.api-ipw.wsmdn.top/",
        }

    ]
}
export { config }