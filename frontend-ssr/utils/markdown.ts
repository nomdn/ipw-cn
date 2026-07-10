import markdownItAnchor from 'markdown-it-anchor'
import GithubSlugger from 'github-slugger'
import { createJavaScriptRegexEngine } from '@shikijs/engine-javascript'
import { fromHighlighter} from '@shikijs/markdown-it/core'
import MarkdownIt from 'markdown-it'
import { createHighlighter } from 'shiki'
const jsEngine = createJavaScriptRegexEngine()
const slugger = new GithubSlugger()

export const md = new MarkdownIt({
  html: true,
  linkify: true,
})
const highlighter = await createHighlighter({
    themes: [
      "vitesse-dark"
    ],
    langs: [
      'text',
      'ts',
      'tsx',
      'js',
      'jsx',
      'vue',
      'json',
      'html',
      'css',
      'scss',
      'bash',
      'shell',
      'yaml',
      'toml',
      'ini',
      'xml',
      'diff',
      'go',
      'cpp',
      'python'
    ],
    engine: jsEngine
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
md.use(fromHighlighter(highlighter,{theme:'vitesse-dark'} as any))
export function renderMarkdown(markdown: string) {
  slugger.reset()
  return md.render(markdown)
}