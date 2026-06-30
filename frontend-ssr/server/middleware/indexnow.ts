// server/middleware/indexnow.ts
export default defineEventHandler((event) => {
  const path = event.path;
  const match = path.match(/^\/([a-f0-9]{32})\.txt$/i);

  if (!match) {
    return; // 不匹配，中间件正常放行，继续走 Nuxt 页面渲染
  }

  const requestedKey = match[1];
  const config = useRuntimeConfig();

  if (requestedKey !== config.indexnowKey) {
    return; // key 不对也直接放行，让它走正常的 404 页面逻辑
  }

  setHeader(event, 'Content-Type', 'text/plain');
  event.node.res.end(config.indexnowKey); // 显式结束响应，避免继续往下传
});