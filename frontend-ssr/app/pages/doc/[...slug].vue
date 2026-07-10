<script setup lang="ts">
import { getDocMeta } from '../../../config/doc'

const route = useRoute();

const path = computed(() => route.path)
const { data: page } = await useFetch(() => `/api/markdown/${path.value}`)
const activeIndex = computed(() => route.path)

const meta = computed(() => getDocMeta(route.path))
useHead({
  title: computed(() => meta.value.title),
  meta: [
    { name: 'description', content: computed(() => meta.value.description) }
  ]
})
</script>
<template>
    <div class="box">
        <el-menu
        :default-active="activeIndex"
        class="sidebar-menu"
        router
    >
            <!-- 1. 独立菜单项 -->
            <el-menu-item index="/doc/usage_docs">
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

        </el-menu>
        <div class="content">
            <div class="markdown-body" v-if="page">
                <div v-html="page" />
            </div>
        </div>
    </div>
</template>
<style scoped>
/* 核心布局容器 */
.box {
    display: flex;
    flex-direction: row; /* 改为 row，实现左右排列 */
    height: 100vh;       /* 占满整个屏幕高度，防止出现页面级滚动条 */
    width: 100%;
}

/* 左侧侧边栏样式 */
.sidebar-menu {
    width: 260px;        /* 固定侧边栏宽度 */
    height: 100%;        /* 高度撑满父容器 */
    flex-shrink: 0;      /* 防止被右侧内容挤压变形 */
    overflow-y: auto;    /* 菜单项过多时，侧边栏内部出现滚动条 */
    border-right: 1px solid #e4e7ed; /* 添加右侧分割线（可选） */
}
.content {
    flex: 1;             /* 右侧内容区域占据剩余空间 */
    padding: 20px;       /* 内边距，避免内容贴边 */
    overflow-y: auto;    /* 内容过多时，右侧区域出现滚动条 */
}
</style>
<style>
@import "github-markdown-css/github-markdown-light.css";
@import "/github-markdown-dark.css";
</style>