
/*
    前端的一系列配置
*/
const config = {
    // Woeker IP查询接口
    v4OnlyAPI: "https://4.wsmdn.dpdns.org/",
    v6OnlyAPI: "https://6.wsmdn.dpdns.org/",
    DualStackAPI: "https://test.wsmdn.dpdns.org/",
    // 后端API地址
    apiBaseUrl: 'https:://api-ipw.wsmdn.dpdns.org/',
    TCPing:{
        DualStack: [

            {
                label: "中国 江苏 移动",
                url:"https:://api-ipw.wsmdn.dpdns.org/"
            },
        ],
        IPv4: [
            
            {
                label: "中国 广州 腾讯云",
                url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
            }
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
                url:"https:://api-ipw.wsmdn.dpdns.org/"
            },
        ],
        IPv4: [
            
            {
                label: "中国 广州 腾讯云",
                url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
            }
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
            url:"https:://api-ipw.wsmdn.dpdns.org/"
        },
        {
            label: "中国 广州 腾讯云",
            url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
        },
            
        {
            label: "中国 四川 沙渠 电信",
            url:"https://cn2-sichuan.api-ipw.wsmdn.top/",
        }

    ]
}
export { config }