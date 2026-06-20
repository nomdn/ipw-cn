import {createRouter,createWebHistory} from 'vue-router'

import IPLocation from './pages/IPLocation.vue'
import SSLTest from './pages/SSLtest.vue'
import IPQuery from './pages/Whatsmyip.vue'
import Webcheck from './pages/Testweb.vue'
import TCPing from './pages/TCPing.vue'
import DNSQuery from './pages/DNSQuery.vue'
import SpeedTest from './pages/WebSpeed.vue'
import IPv4SpeedTest from './pages/ipv4/IPv4speedtest.vue'
import IPv4TCPing from './pages/ipv4/IPv4TCPing.vue'
const routes=[
    {path:'/',component:IPQuery},
    {path:'/ssl',component:SSLTest},
    {path:'/ipv6',component:IPLocation},
    {path:'/ipv6webcheck',component:Webcheck},
    {path:'/ipv6tcping',component:TCPing},
    {path:'/dns',component:DNSQuery},
    {path:'/ipv6speedtest',component:SpeedTest},
    {path:'/speedtest',component:IPv4SpeedTest},
    {path:'/tcping',component:IPv4TCPing}

]

export const router=createRouter({
    history:createWebHistory(),
    routes
})

export default router