import { createHighlighter } from 'shiki'
export const highlighter = await createHighlighter({
  themes: [
    'github-light',
    'github-dark'
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
]
})
export async function highlightCode(code: string, lang: string) {
  try {
    return await highlighter.codeToHtml(code, { lang: 'bash', theme:'github-dark' })
  } catch (error) {
    console.error('Error highlighting code:', error)
    return code
  }
}