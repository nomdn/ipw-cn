// server/routes/[key].txt.ts
export default defineEventHandler((event) => {
  const config = useRuntimeConfig();
  const requestedKey = getRouterParam(event, 'key');

  if (requestedKey !== config.indexnowKey) {
    throw createError({ statusCode: 404, statusMessage: 'Not Found' });
  }

  setHeader(event, 'Content-Type', 'text/plain');
  return config.indexnowKey;
});