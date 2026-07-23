import markdownItAnchor from 'markdown-it-anchor'
import GithubSlugger from 'github-slugger'
import { highlighter } from './shiki'
import { fromHighlighter } from '@shikijs/markdown-it/core'
import MarkdownIt from 'markdown-it'

const slugger = new GithubSlugger()
export const md = new MarkdownIt({
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
md.use(fromHighlighter(highlighter,{theme:'vitesse-dark'} as any))
export function renderMarkdown(markdown: string) {
  slugger.reset()
  return md.render(markdown)
}