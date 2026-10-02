import { createContext, useContext } from 'react'
import type { Lang } from './i18n/dict'
import { en, id } from './i18n/dict'

export const LangCtx = createContext<{ lang: Lang; setLang: (l: Lang) => void; t: (k: string) => string }>({
  lang: 'en',
  setLang: () => {},
  t: (k: string) => en[k] ?? k,
})

export const useLang = () => useContext(LangCtx)

export function translate(lang: Lang, k: string): string {
  return lang === 'id' ? (id[k] ?? en[k] ?? k) : (en[k] ?? k)
}
