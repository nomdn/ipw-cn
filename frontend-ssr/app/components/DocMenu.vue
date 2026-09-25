<script setup lang="ts">
/**
 * 文档导航菜单（全站唯一实现，结构别在别处再抄一份）
 *
 * - 宽屏：由 /doc 与 /doc/** 页面直接渲染成左侧固定侧边栏（由页面传 class="sidebar-menu"）
 * - 窄屏：页面那边把侧边栏整个隐藏，改由根布局的左侧抽屉菜单挂同一份本组件，
 *   这样手机上不需要两个菜单，也不至于没入口进文档。
 */
const route = useRoute()
const activeIndex = computed(() => route.path)

// 所有可展开分组的 index（含嵌套的「最佳实践」）。
// 抽屉里给展开态用：手机上没有 hover，分组默认收着的话用户还得先点开一层才能看到目的地。
const ALL_GROUPS = [
  'IPv6 用户端',
  '最佳实践',
  '云服务器配置 IPv6',
  'IPv6 协议规范',
  'API 接口',
]

const props = withDefaults(defineProps<{
  /** true = 所有分组初始展开（抽屉用）；false = 保持 element-plus 默认的收起态（页面侧栏用） */
  expandAll?: boolean
}>(), { expandAll: false })

// undefined 让 element-plus 走自己的默认值，宽屏下行为与改造前完全一致
const openeds = computed(() => (props.expandAll ? ALL_GROUPS : undefined))

const emit = defineEmits<{ select: [index: string] }>()
</script>
<template>
  <el-menu
    :default-active="activeIndex"
    :default-openeds="openeds"
    router
    @select="(index: string) => emit('select', index)"
  >
    <!-- 1. 独立菜单项 -->
    <el-menu-item index="/doc">
      <span>IPv6 工具箱使用文档</span>
    </el-menu-item>

    <!-- 2. IPv6 用户端 -->
    <el-sub-menu index="IPv6 用户端">
      <template #title>
        <span>IPv6 用户端</span>
      </template>

      <el-menu-item index="/doc/user/enable_ipv6">个人宽带如何开启IPv6网络访问</el-menu-item>
      <el-menu-item index="/doc/user/cmd_bash_disable_ipv6">命令行禁用/启用IPv6本地网络</el-menu-item>
      <el-menu-item index="/doc/user/cmd_getip">命令行(curl)获取 IPv4 和 IPv6 地址</el-menu-item>
      <el-menu-item index="/doc/user/view_ipv6_adress_url">浏览器访问 IPv6 地址</el-menu-item>
      <el-menu-item index="/doc/user/ipv4_ipv6_prefix_precedence">Windows 10/11 设置 IPv4/IPv6 访问优先级</el-menu-item>
      <el-menu-item index="/doc/user/ipv6_daohang">国内 IPv6 资源导航</el-menu-item>
      <el-menu-item index="/doc/user/pure_ipv6_website">国内纯 IPv6 网站导航</el-menu-item>
      <el-menu-item index="/doc/user/dns">全国各省 DNS 服务器列表</el-menu-item>
      <el-menu-item index="/doc/user/ipv6_dns">IPv6 DNS 地址列表</el-menu-item>
      <el-menu-item index="/doc/user/code_getip">Python/Go 获取 IPv4 和 IPv6 地址</el-menu-item>
      <el-menu-item index="/doc/user/dns_dig">DNS 解析流程</el-menu-item>

      <!-- 嵌套子菜单：最佳实践 -->
      <el-sub-menu index="最佳实践">
        <template #title>
          <span>最佳实践</span>
        </template>
        <el-menu-item index="/doc/user/AliyunAuthorizeSecurityGroup">阿里云自动化添加安全组</el-menu-item>
        <el-menu-item index="/doc/user/TencentCloudAddSecurityGroup">腾讯云自动化添加安全组</el-menu-item>
      </el-sub-menu>
    </el-sub-menu>

    <!-- 3. 云服务器配置 IPv6 -->
    <el-sub-menu index="云服务器配置 IPv6">
      <template #title>
        <span>云服务器配置 IPv6</span>
      </template>
      <el-menu-item index="/doc/server/website_enable_ipv6">网站开启 IPv6 的三种方式</el-menu-item>
      <el-menu-item index="/doc/server/tencent_cloud_cvm_ipv6">腾讯云 cvm 开启 IPv6</el-menu-item>
      <el-menu-item index="/doc/server/nginx_ipv6">Nginx 开启 IPv6</el-menu-item>
      <el-menu-item index="/doc/server/ipv6webcheck">如何确认一个网站是否开启 IPv6</el-menu-item>
      <el-menu-item index="/doc/server/ipv6_sign">网站增加支持IPv6访问标识</el-menu-item>
      <el-menu-item index="/doc/server/ipv6_domain_record">如何为域名添加 IPv6 解析记录</el-menu-item>
      <el-menu-item index="/doc/server/http2">网站如何开启 HTTP/2</el-menu-item>
      <el-menu-item index="/doc/server/check_userip_ipv4_ipv6">JS 检查网络是 IPv4 还是 IPv6</el-menu-item>
      <el-menu-item index="/doc/server/tls">网站如何开启 TLS</el-menu-item>
    </el-sub-menu>

    <!-- 4. IPv6 协议规范 -->
    <el-sub-menu index="IPv6 协议规范">
      <template #title>
        <span>IPv6 协议规范</span>
      </template>
      <el-menu-item index="/doc/rfc/rfc8200">IPv6 RFC8200 解读</el-menu-item>
      <el-menu-item index="/doc/rfc/ipv6_address_format">IPv6 地址标识方法</el-menu-item>
      <el-menu-item index="/doc/user/tcpdump_ipv6">tcpdump 分析IPv6包</el-menu-item>
      <el-menu-item index="/doc/user/wireshark_ipv6">WireShark 分析IPv6包头</el-menu-item>
      <el-menu-item index="/doc/user/ipv6_ping">IPv6 Ping 原理</el-menu-item>
    </el-sub-menu>

    <!-- 5. API 接口 -->
    <el-sub-menu index="API 接口">
      <template #title>
        <span>API 接口</span>
      </template>
      <el-menu-item index="/doc/api/ip/locate">获取客户端公网 IP 及位置</el-menu-item>
      <el-menu-item index="/doc/api/ip/myip">获取客户端公网 IP 地址</el-menu-item>
      <el-menu-item index="/doc/api/ip/query">查询指定 IP 地址的位置信息</el-menu-item>
      <el-menu-item index="/doc/api/whois/query">查询域名的 WHOIS 信息</el-menu-item>
    </el-sub-menu>
  </el-menu>
</template>
