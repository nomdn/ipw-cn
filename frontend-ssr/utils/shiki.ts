import { createHighlighter } from 'shiki'
import { createJavaScriptRegexEngine } from '@shikijs/engine-javascript'
const jsEngine = createJavaScriptRegexEngine()
export const highlighter = await createHighlighter({
  themes: ['github-light', 'github-dark', 'vitesse-dark'],
  langs: [
    'text', 'ts', 'tsx', 'js', 'jsx', 'vue', 'json', 'html',
    'css', 'scss', 'bash', 'shell', 'yaml', 'toml', 'ini',
    'xml', 'diff', 'go', 'cpp', 'python'
  ],
  engine: jsEngine
})
export async function highlightCode(code: string, lang: string) {
  try {
    return await highlighter.codeToHtml(code, { lang, theme: 'github-dark' })
  } catch (error) {
    console.error('Error highlighting code:', error)
    return code
  }
}