import {
  defineEventHandler,
  createError
} from 'h3'

import { renderMarkdown } from '../../../utils/markdown'


export default defineEventHandler(async (event) => {

  const params = event.context.params?.path


  if (!params) {
    throw createError({
      statusCode: 400,
      statusMessage: 'Missing markdown path'
    })
  }


  // [...path] 返回字符串数组
  const path = Array.isArray(params)
    ? params.join('/')
    : params


  // 防止目录穿越
  if (
    path.includes('..') ||
    path.includes('\\')
  ) {
    throw createError({
      statusCode: 400,
      statusMessage: 'Invalid path'
    })
  }


  const filePath = `${path}.md`

const markdown =
  await useStorage('assets:server')
    .getItem<string>(
      `markdown/${path}.md`
    )


  if (!markdown) {
    throw createError({
      statusCode: 404,
      statusMessage: 'Markdown not found'
    })
  }


  const html = await renderMarkdown(markdown)


  return html

})