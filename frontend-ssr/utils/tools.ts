export function extractHost(url: string): string {
  const regex = /^(?:[a-zA-Z][a-zA-Z\d+.-]*:\/\/)?(?:[^\s@/]+@)?(?<host>(?:\[(?:[0-9a-fA-F:]+)\]|(?:\d{1,3}(?:\.\d{1,3}){3})|(?:[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?(?:\.[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?)*))(?::\d{1,5})?)(?:[/?#][^\s]*)?$/u;

  const match = url.trim().match(regex);
  return match?.groups?.host ?? url;
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
