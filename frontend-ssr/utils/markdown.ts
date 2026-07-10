import MarkdownIt from 'markdown-it'
import markdownItAnchor from 'markdown-it-anchor'
import GithubSlugger from 'github-slugger'
import { highlighter } from './shiki'

const slugger = new GithubSlugger()

export const md = new MarkdownIt({
  html: true,
  linkify: true,

  highlight(code, lang) {
    const language =
      lang &&
      highlighter.getLoadedLanguages().includes(lang as any)
        ? lang
        : 'text'

    return highlighter.codeToHtml(code, {
      lang: language,
      theme: 'github-dark'
    })
  }
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
export function renderMarkdown(markdown: string) {
  slugger.reset()
  return md.render(markdown)
}