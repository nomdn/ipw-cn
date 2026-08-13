
/*
    前端的一系列配置
*/
const config = {
    siteUrl: "https://ipw.wsmdn.top/",
    // 外部独立中间件服务列表（base URL）。
    // 与前端自带中间件（server/routes/middleware/[...slug].get.ts 本地转发）同级，
    // 前端 /middleware/* 请求会依次尝试这些节点（出错重试下一个），
    // 前端自带中间件始终放在候选列表最后一位兜底。
    // 未配置或为空数组时，直接使用前端自带中间件。
    Middleware: <string[]>[
        "http://127.0.0.1:8091/",
        // "https://middleware-2.wsmdn.top/",
    ],
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
    apiBaseUrls: [
        {
            label: "中国 江苏 移动",
            id: "cn-jiangsu",
            url: "https://cn-jiangsu.api-ipw.wsmdn.top/"
        },
        {
            label: "中国 广东 深圳 龙岗 坪地街道 中国移动",
            id: "cn-shenzhen",
            url: "https://cn-shenzhen.api-ipw.wsmdn.top/"
        }
    ],
    IPLocationAPIs: [
        {
            label: "中国 江苏 移动",
            id: "cn-jiangsu",
            url: "https://cn-jiangsu.api-ipw.wsmdn.top/"
        },
        {
            label: "中国 四川 沙渠 电信[ZFC]",
            id: "cn2-sichuan",
            url: "https://cn2-sichuan.api-ipw.wsmdn.top/"
        }
    ],
    // 全站是否禁止搜索引擎索引
    noindex: false,
    TCPing:{
        DualStack: [
            {
                label: "中国 江苏 移动",
                id: "cn-jiangsu",
                url :"https://cn-jiangsu.api-ipw.wsmdn.top/",
            },
            {
                label: "中国 广东 深圳 龙岗 坪地街道 中国移动",
                id: "cn-shenzhen",
                url :"https://cn-shenzhen.api-ipw.wsmdn.top/",
            },
        ],
        IPv4: [
            {
                label: "中国 广东 广州 腾讯云",
                id: "cn-guangzhou",
                url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
            },
            {
                label: "新加坡 腾讯云",
                id: "sg-1",
                url :"https://sg-1.api-ipw.wsmdn.top/",
            },
            {
                label: "中国 香港 新界 西贡区 将军澳 MICC[ZFC]",
                id: "hk-shatin",
                url :"https://china-hk-kowloon-shatindistrict-newcloud.ipwapi.zfap.wsmdn.top/",
            },
            {
                label: "中国 陕西 西安 北经济技术开发区 未央区凤城 中国电信[ZFC]",
                id: "cn-xian",
                url :"https://cn-xian-shaanxiprovince.api-ipw.wsmdn.top/",
            },
            {
                label: "美国/加利福尼亚州/洛杉矶/蒙特雷帕克/USCD[ZFC]",
                id: "us-la",
                url :"https://america-california-losangeles.ipwapi.zfap.wsmdn.top/",
            }
        ],
        IPv6: [
            {
                label: "中国 四川 沙渠 电信[ZFC]",
                id: "cn2-sichuan",
                url:"https://cn2-sichuan.api-ipw.wsmdn.top/",
            },
            {
                label: "中国 香港 九龙城区 旺角东 Cloudie[ZFC]",
                id: "lntl-cn-hk-kowloon",
                url:"https://lntl-cn-hk-kowloon.api-ipw.wsmdn.top/",
            }
        ]
    },
    SpeedTest:{
        DualStack: [
            {
                label: "中国 江苏 移动",
                id: "cn-jiangsu",
                url :"https://cn-jiangsu.api-ipw.wsmdn.top/",
            },
            {
                label: "中国 广东 深圳 龙岗 坪地街道 中国移动",
                id: "cn-shenzhen",
                url :"https://cn-shenzhen.api-ipw.wsmdn.top/",
            },
        ],
        IPv4: [
            {
                label: "中国 广东 广州 腾讯云",
                id: "cn-guangzhou",
                url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
            },
            {
                label: "新加坡 腾讯云",
                id: "sg-1",
                url :"https://sg-1.api-ipw.wsmdn.top/",
            },
            {
                label: "中国 香港 新界 西贡区 将军澳 MICC[ZFC]",
                id: "hk-shatin",
                url :"https://china-hk-kowloon-shatindistrict-newcloud.ipwapi.zfap.wsmdn.top/",
            },
            {
                label: "中国 陕西 西安 北经济技术开发区 未央区凤城 中国电信[ZFC]",
                id: "cn-xian",
                url :"https://cn-xian-shaanxiprovince.api-ipw.wsmdn.top/",
            },
            {
                label: "美国/加利福尼亚州/洛杉矶/蒙特雷帕克/USCD[ZFC]",
                id: "us-la",
                url :"https://america-california-losangeles.ipwapi.zfap.wsmdn.top/",
            }
        ],
        IPv6: [
            {
                label: "中国 四川 沙渠 电信[ZFC]",
                id: "cn2-sichuan",
                url:"https://cn2-sichuan.api-ipw.wsmdn.top/",
            },
            {
                label: "中国 香港 九龙城区 旺角东 Cloudie[ZFC]",
                id: "lntl-cn-hk-kowloon",
                url:"https://lntl-cn-hk-kowloon.api-ipw.wsmdn.top/",
            }
        ]
    },
    NSLookup:[
        {
            label: "中国 江苏 移动",
            id: "cn-jiangsu",
            url :"https://cn-jiangsu.api-ipw.wsmdn.top/",
        },
        {
            label: "中国 广州 腾讯云",
            id: "cn-guangzhou",
            url :"https://cn-guangzhou.api-ipw.wsmdn.top/",
        },
        {
            label: "新加坡 腾讯云",
            id: "sg-1",
            url :"https://sg-1.api-ipw.wsmdn.top/",
        },
        {
            label: "中国 四川 沙渠 电信[ZFC]",
            id: "cn2-sichuan",
            url:"https://cn2-sichuan.api-ipw.wsmdn.top/",
        },
        {
            label: "中国 陕西 西安 北经济技术开发区 未央区凤城 中国电信[ZFC]",
            id: "cn-xian",
            url :"https://cn-xian-shaanxiprovince.api-ipw.wsmdn.top/",
        },
        {
            label: "中国 香港 新界 西贡区 将军澳 MICC[ZFC]",
            id: "hk-shatin",
            url :"https://china-hk-kowloon-shatindistrict-newcloud.ipwapi.zfap.wsmdn.top/",
        },
        {
            label: "中国 香港 九龙城区 旺角东 Cloudie[ZFC]",
            id: "lntl-cn-hk-kowloon",
            url:"https://lntl-cn-hk-kowloon.api-ipw.wsmdn.top/",
        },
        {
            label: "中国 广东 深圳 龙岗 坪地街道 中国移动",
            id: "cn-shenzhen",
            url :"https://cn-shenzhen.api-ipw.wsmdn.top/",
        },
        {
            label: "美国/加利福尼亚州/洛杉矶/蒙特雷帕克/USCD[ZFC]",
            id: "us-la",
            url :"https://america-california-losangeles.ipwapi.zfap.wsmdn.top/",
        }
    ]
}
export { config }
