
/*
    前端的一系列配置
*/

// 是否开发环境。
// 本文件同时被 nuxt.config 与 nitro server 引用，那两处 import.meta.dev 尚未注入（undefined），
// 因此只有严格为 true 才算 dev；应用代码里由 Vite 的 define 替换成真实布尔值。
const __DEV__ = (import.meta as any).dev === true

/**
 * 本机调试节点：仅开发环境注入，对应 middleware-go/setting.json 里的 local
 * （url 指向本机节点，默认 127.0.0.1:18081）。生产构建为空数组。
 *
 * 位置约定：调试节点放**第一优先级**——各页把 `...devNodes` 拼在节点列表最前面；
 * 本机只有 IPv4，所以 IPv6 页（ipv6tcping / ipv6speedtest）不拼它。
 */
export const devNodes: { label: string; id: string; url: string }[] = __DEV__
    ? [{ label: "本机 127.0.0.1:18081（调试）", id: "local", url: "" }]
    : []

const config = {
    siteUrl: "https://nipw.cn/",
    // 站点名称：用于页面标题 / 描述 / 页脚品牌展示
    siteName: "柠檬味ipw.cn",
    // 是否启用前端内置中间件（server/routes/middleware/[...slug].get.ts 本地转发）。
    // true（默认）：/middleware/* 候选为「外部节点… + 内置中间件（最后一位兜底）」；
    // false：仅使用外部节点，候选列表不含内置中间件。
    EnableInternalMiddleware: true,
    // 内置中间件单 IP 限流次数（次/分钟）；0 表示不限流。默认 120。
    rateLimitPerMinute: 120,
    // 外部独立中间件服务列表（base URL）。
    // 与前端自带中间件（server/routes/middleware/[...slug].get.ts 本地转发）同级，
    // 前端 /middleware/* 请求会依次尝试这些节点（出错重试下一个），
    // 前端自带中间件始终放在候选列表最后一位兜底。
    // 未配置或为空数组时，直接使用前端自带中间件。
    Middleware: <string[]>[
        // dev：本机 middleware-go（8091）排最前 —— 调试节点 id "local" 只有它认识，
        // 远端中间层拿到会直接返回 Invalid backend ID（400，不在重试码里）。
        ...(__DEV__ ? ["http://127.0.0.1:8091/"] : []),
        "https://middleware-1.api-ipw.wsmdn.top/"
        // "https://middleware-2.wsmdn.top/",
    ],
    // Umami 统计
    umamiHost: "https://umami.wsmdn.top/",
    umamiScriptUrl: "https://umami.wsmdn.top/zako.js",
    umamiWebsiteId: "69a91329-b110-4cf7-a04a-be4360b1a8d3",
    // 中华人民共和国备案系统
    ICP: "",
    GongAn: "",
    // 全站是否禁止搜索引擎索引
    noindex: false,
    // Worker IP查询接口
    v4OnlyAPI: "https://4.wsmdn.top",
    v6OnlyAPI: "https://6.wsmdn.top",
    DualStackAPI: "https://test.wsmdn.top",
    // ---- 节点池结构 ----
    // IPLocationAPI：IP 归属查询（location / asn）上游节点池（纯数组，无栈区分）
    IPLocationAPI: [
        {
            label: "中国 江苏 移动",
            id: "cn-jiangsu",
            url: "https://cn-jiangsu.api-ipw.wsmdn.top/"
        },
        // {
        //     label: "中国 湖北 武汉 电信",
        //     id: "cn-wuhan-chinatelecom",
        //     url: ""
        // },
        {
            label: "中国 四川 沙渠 电信[ZFC]",
            id: "cn2-sichuan",
            url: "https://cn2-sichuan.api-ipw.wsmdn.top/"
        }
    ],
    // APIBaseURL：其余拨测（whois / ssl / detail / dns / dnssec / tcping / speed）上游节点池，含 { IPv6, IPv4, DualStack } 三栈
    APIBaseURL: {
        IPv6: [
            {
                label: "中国 四川 沙渠 电信[ZFC]",
                id: "cn2-sichuan",
                url: "https://cn2-sichuan.api-ipw.wsmdn.top/"
            },
            {
                label: "中国 香港 九龙城区 旺角东 Cloudie[ZFC]",
                id: "lntl-cn-hk-kowloon",
                url: "https://lntl-cn-hk-kowloon.api-ipw.wsmdn.top/"
            },
        ],
        IPv4: [
            {
                label: "中国 广东 广州 腾讯云",
                id: "cn-guangzhou",
                url: "https://cn-guangzhou.api-ipw.wsmdn.top/"
            },
            {
                label: "新加坡 腾讯云",
                id: "sg-1",
                url: "https://sg-1.api-ipw.wsmdn.top/"
            },
            {
                label: "中国 香港 新界 西贡区 将军澳 MICC[ZFC]",
                id: "hk-shatin",
                url: "https://china-hk-kowloon-shatindistrict-newcloud.ipwapi.zfap.wsmdn.top/"
            },
            {
                label: "中国 陕西 西安 北经济技术开发区 未央区凤城 中国电信[ZFC]",
                id: "cn-xian",
                url: "https://cn-xian-shaanxiprovince.api-ipw.wsmdn.top/"
            },
            {
                label: "美国/加利福尼亚州/洛杉矶/蒙特利奇帕克/USCD[ZFC]",
                id: "us-la",
                url: "https://america-california-losangeles.ipwapi.zfap.wsmdn.top/"
            },
            // IP 直连新节点：URL 留空，仅经独立中间件转发
            {
                label: "山东 枣庄 双线",
                id: "zaozhuang",
                url: ""
            },
            {
                label: "香港 Cogent",
                id: "hongkong",
                url: ""
            },
            // // WS 通道节点：URL 留空，拨测经中间件 WS 转发
            // {
            //     label: "呼和浩特移动",
            //     id: "2faa15e6-18a7-4355-b422-ca53277d8d77",
            //     url: ""
            // }
        ],
        DualStack: [

            {
                label: "中国 江苏 移动",
                id: "cn-jiangsu",
                url: "https://cn-jiangsu.api-ipw.wsmdn.top/"
            },
            {
                label: "中国 广东 深圳 龙岗 坪地街道 中国移动",
                id: "cn-shenzhen",
                url: "https://cn-shenzhen.api-ipw.wsmdn.top/"
            },
            {
                label: "河南 安阳 联通",
                "id": "anyang-cu"
            }
            // {
            //     label: "中国 湖北 武汉 电信",
            //     id: "cn-wuhan-chinatelecom",
            //     url: ""
            // },
        ]
    }
}
export { config }
