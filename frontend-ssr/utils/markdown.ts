import markdownItAnchor from 'markdown-it-anchor'
import GithubSlugger from 'github-slugger'
import { getSingletonHighlighter } from './shiki.bundle'
import { fromHighlighter } from '@shikijs/markdown-it/core'
import MarkdownIt from 'markdown-it'
import { config } from '../config/index'

const slugger = new GithubSlugger()

let md: MarkdownIt | null = null
let initPromise: Promise<void> | null = null

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
  // 文档配图指向静态资产加速域：cdnURL 只会改写 Nuxt 自己管理的资产引用，
  // 这里是 markdown 文本渲染出的 <img>，不会被自动加前缀，须手动拼。
  // 只改站内根路径 /docimg/**，其余（外链、data:）原样保留。
  const defaultImage = md.renderer.rules.image!
  md.renderer.rules.image = (tokens, idx, options, env, self) => {
    const token = tokens[idx]
    const src = token.attrGet('src')
    if (src && src.startsWith('/docimg/')) {
      token.attrSet('src', config.assetOrigin + src)
    }
    return defaultImage(tokens, idx, options, env, self)
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
