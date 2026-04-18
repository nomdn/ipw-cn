import {createRouter,createWebHistory} from 'vue-router'

import IPLocation from './pages/IPLocation.vue'
import SSLTest from './pages/SSLtest.vue'
import IPQuery from './pages/Whatsmyip.vue'

const routes=[
    {path:'/',component:IPQuery},
    {path:'/ssl',component:SSLTest},
    {path:'/ipv6',component:IPLocation}
]

export const router=createRouter({
    history:createWebHistory(),
    routes
})

export default router