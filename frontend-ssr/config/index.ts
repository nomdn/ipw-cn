
/*
    前端的一系列配置
*/
const config = {
    siteUrl: "https://ipw.wsmdn.top/",
    // Umami 统计
    umamiHost: "https://umami.wsmdn.top/",
    umamiScriptUrl: "https://umami.wsmdn.top/zako.js",
    umamiWebsiteId: "69a91329-b110-4cf7-a04a-be4360b1a8d3",
    // 中华人民共和国备案系统
    ICP: "苏ICP备2026012471号",
    GongAn: "苏公网安备32132402000813号",
    // Worker IP查询接口
    v4OnlyAPI: "https://4.wsmdn.dpdns.org/",
    v6OnlyAPI: "https://6.wsmdn.dpdns.org/",
    DualStackAPI: "https://test.wsmdn.dpdns.org/",
    // 后端API地址
    apiBaseUrl: 'https://cn-jiangsu.api-ipw.wsmdn.top/',
    IPLocationAPI : "https://cn2-sichuan.api-ipw.wsmdn.top/",
    // 全站是否禁止搜索引擎索引
    noindex: false,
    TCPing:{
        DualStack: [
            {
                label: "中国 江苏 移动",
                url :"https://cn-jiangsu.api-ipw.wsmdn.top/",
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
                label: "中国 江苏 移动",
                url :"https://cn-jiangsu.api-ipw.wsmdn.top/",
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
            label: "中国 江苏 移动",
            url :"https://cn-jiangsu.api-ipw.wsmdn.top/",
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
