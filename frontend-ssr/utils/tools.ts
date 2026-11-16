export function extractHost(url: string): string {
  const regex = /^(?:[a-zA-Z][a-zA-Z\d+.-]*:\/\/)?(?:[^\s@/]+@)?(?<host>(?:\[(?:[0-9a-fA-F:]+)\]|(?:\d{1,3}(?:\.\d{1,3}){3})|(?:[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?(?:\.[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?)*))(?::\d{1,5})?)(?:[/?#][^\s]*)?$/u;

  const match = url.trim().match(regex);
  return match?.groups?.host ?? url;
}

/**
 * extractTarget 只剥掉协议前缀，保留 host 之后全部内容（路径 / 查询串）。
 *
 * 供 detail（网站检测）与 speed（测速）这类「目标是完整 URL」的拨测使用：
 * 之前沿用 extractHost 会把路径整个丢掉，用户测 https://example.com/bigfile.zip
 * 实际测成了首页，测速/检测结论都与输入对不上。
 *
 * 注意分界：tcping 仍用 extractHost —— TCP 探测目标只能是 host:port，
 * 路径没有语义；whois 输入本来就是纯域名。
 */
export function extractTarget(url: string): string {
  return url.trim().replace(/^[a-zA-Z][a-zA-Z\d+.-]*:\/\//, '');
}

export function formatTime(ms?: number | null): string {
  if (ms == null || typeof ms !== 'number' || !isFinite(ms) || ms <= 0) return '-'
  if (ms < 1000) {
    return `${ms.toFixed(2)} ms`
  }
  return `${(ms / 1000).toFixed(2)} s`
}

export function formatSpeed(speed?: number | null): string {
  if (speed == null || typeof speed !== 'number' || !isFinite(speed)) return '-'
  return `${speed.toFixed(2)} KB/s`
}

export function formatSize(bytes?: number | null): string {
  if (bytes == null || typeof bytes !== 'number' || !isFinite(bytes)) return '-'
  if (bytes < 1024) {
    return `${bytes} B`
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(2)} KB`
  }
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

export function getStatusCodeClass(code?: number | null): string {
  if (code == null || typeof code !== 'number') return 'status-error'
  if (code >= 200 && code < 300) return 'status-success'
  if (code >= 300 && code < 400) return 'status-warning'
  return 'status-error'
}

export function isIPv4(ip: string): boolean {
  const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
  return ipRegex.test(ip);
}

/**
 * 拨测目标写入 /middleware/<id>/<type>/<target> 路径专用编码。
 *
 * 单层 encodeURIComponent 不够：h3 解析请求 URL 时会先做一次百分号解码、
 * 再按 '?' 拆查询串（h3 1.15 实测），目标 URL 自带的查询串会被当成拨测请求
 * 自己的 query 截掉（进 getQuery 后与高级选项混在一起，无法区分）。
 * 所以 '?' 再叠一层编码（%3F → %253F），穿过 h3 后由代理端统一解码还原，
 * 再按段重编码发给节点。'/' 与 '#' 单层编码即可穿过，无需处理。
 */
export function encodeTargetPath(target: string): string {
  return encodeURIComponent(target).replace(/%3F/gi, '%253F')
}

/**
 * 把参数对象拼到 URL 上（空值跳过，自动判断用 ? 还是 &）。
 * 用于高级选项：dns4/dns6/resolve/resolve4/resolve6 这些都是「请求级解析参数」，
 * 只有用户填了才下发，没填时 URL 与加参数之前逐字节一致（缓存键也不变）。
 */
export function withQuery(url: string, params: Record<string, string>): string {
  const pairs: string[] = []
  for (const [key, value] of Object.entries(params)) {
    if (value) pairs.push(encodeURIComponent(key) + '=' + encodeURIComponent(value))
  }
  if (pairs.length === 0) return url
  return url + (url.includes('?') ? '&' : '?') + pairs.join('&')
}
