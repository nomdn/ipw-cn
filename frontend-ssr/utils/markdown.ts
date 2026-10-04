import markdownItAnchor from 'markdown-it-anchor'
import GithubSlugger from 'github-slugger'
import { getSingletonHighlighter } from './shiki.bundle'
import { fromHighlighter } from '@shikijs/markdown-it/core'
import MarkdownIt from 'markdown-it'

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
