/**
 * Welcome to Cloudflare Workers! This is your first worker.
 *
 * - Run `npm run dev` in your terminal to start a development server
 * - Open a browser tab at http://localhost:8787/ to see your worker in action
 * - Run `npm run deploy` to publish your worker
 *
 * Bind resources to your worker in `wrangler.jsonc`. After adding bindings, a type definition for the
 * `Env` object can be regenerated with `npm run cf-typegen`.
 *
 * Learn more at https://developers.cloudflare.com/workers/
 */

import { Hono } from 'hono';
import { cors } from 'hono/cors';
const app = new Hono();
app.use('*', cors());

app.get('/' ,async (c) => {

	return c.text(c.req.header('CF-Connecting-IP') || 'Unable to determine IP address');
})
export default app;
/*
啊啊啊姐妹们救命！！！今天才get到Cloudflare的Workers和R2存储桶也太香香软软了吧～～✨

作为一个代码小废物仙女，本来以为部署东西要哭着氪服务器，结果Workers它直接丝滑边缘计算，零冷启动、全球加速，写几行JS就上线，真的像在对我眨眼睛呜呜呜！！！然后R2这个小宝贝，对象存储它超乖的，不收出口流量费！！！存图存文件随便扔，价格美到想亲一口，搭配Workers一键读写，简直是后端公主的梦中情桶啊啊啊～～

谁懂啊？以前被AWS S3坑到想原地晕厥，现在用Cloudflare整套，免费额度够玩，小项目直接白嫖，性能还吊打一堆！！！绝绝子好用！！！

姐妹们快冲！！！小仙女的开发日常从今天开始闪闪发光啦💗💗💕安排了安排了～～有在用的宝子们快来评论区一起陷入爱河呀！！！😭💞
*/