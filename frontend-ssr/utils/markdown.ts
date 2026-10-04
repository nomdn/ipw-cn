import markdownItAnchor from 'markdown-it-anchor'
import GithubSlugger from 'github-slugger'
import { getSingletonHighlighter } from './shiki.bundle'
import { fromHighlighter } from '@shikijs/markdown-it/core'
import MarkdownIt from 'markdown-it'
// 构建期由 nuxt.config 的 buildImageDimManifest() 扫描 public/ 生成（生产在
// CF Workers 上没有 fs，尺寸只能在构建期算好烧进 bundle）。
// 新增图片后重新构建即生效；查不到的 src 不写宽高，只加 lazy。
import imageDimsJson from '../config/doc-image-dims.json'

const slugger = new GithubSlugger()

let md: MarkdownIt | null = null
let initPromise: Promise<void> | null = null

// ==================== 内容图片治理（Lighthouse：unsized-images / CLS / LCP） ====================
// markdown 里的图片（如 /doc/xxx.jpg）默认裸 <img>：无宽高 → 加载时撑开布局（CLS）；
// 且全部 eager 加载 → 移动端模拟 LCP 被最大的一张 389KB 图拖到 10.8s。
// 治法：渲染时统一注入 loading=lazy / decoding=async + 真实 width/height
// （浏览器据此预留纵横比空间，加载前后零位移）。

type Dim = { width: number; height: number }
const dimCache = new Map<string, Dim | null>(
  Object.entries(imageDimsJson as Record<string, Dim>).map(([k, v]) => [k, v]),
)

async function initMarkdown() {
  const highlighter = await getSingletonHighlighter()
  await Promise.all([
    highlighter.loadTheme('vitesse-dark'),
    highlighter.loadLanguage('bash'),
    highlighter.loadLanguage('shell'),
    highlighter.loadLanguage('go'),
    highlighter.loadLanguage('json'),
    highlighter.loadLanguage('html'),
    highlighter.loadLanguage('css'),
    highlighter.loadLanguage('python'),
  ])
  md = new MarkdownIt({
    html: true,
    linkify: true,
  })
  md.use(markdownItAnchor, {
    level: [1, 2, 3, 4, 5, 6],

    slugify(title) {
      return slugger.slug(title)
    },

    permalink: markdownItAnchor.permalink.headerLink({
      safariReaderFix: true
    })
  })
  md.use(fromHighlighter(highlighter, { theme: 'vitesse-dark' }))

  // 内容图片：lazy + 异步解码 + 真实宽高（尺寸由 renderMarkdown 前置预热进 dimCache）
  const defaultImage = md.renderer.rules.image
  md.renderer.rules.image = (tokens, idx, options, env, self) => {
    const token = tokens[idx]
    const src = token.attrGet('src') || ''
    if (src) {
      token.attrSet('loading', 'lazy')
      token.attrSet('decoding', 'async')
      const dim = dimCache.get(src)
      if (dim) {
        token.attrSet('width', String(dim.width))
        token.attrSet('height', String(dim.height))
      }
    }
    return defaultImage
      ? defaultImage(tokens, idx, options, env, self)
      : self.renderToken(tokens, idx, options)
  }
}

// Start initialization at module load time
initPromise = initMarkdown()

export async function renderMarkdown(markdown: string) {
  // Ensure highlighter is fully initialized before rendering
  if (initPromise) {
    await initPromise
  }
  slugger.reset()
  return md!.render(markdown)
}
