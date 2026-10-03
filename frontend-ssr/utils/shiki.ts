import { codeToHtml } from './shiki.bundle'

export async function highlightCode(code: string, lang: string) {
  try {
    return await codeToHtml(code, {
      lang,
      theme: 'github-dark',
      // github-dark 的注释色 #6A737D 在它自带底色 #24292E 上只有 3.04:1（axe 的 color-contrast，
      // 首页 12 条里有 5 条都是它）。用 shiki 原生的颜色替换提亮到 #8B949E = 4.77:1 ——
      // 只动这一个 token，换主题（比如已在 bundle 里的 vitesse-dark）会把整块代码的配色一起换掉。
      colorReplacements: { '#6a737d': '#8b949e' },
    })
  } catch (error) {
    console.error('Error highlighting code:', error)
    return code
  }
}
