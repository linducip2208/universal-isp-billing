import { createContext, useContext } from 'react'
import type { Lang } from './i18n/dict'
import { ar, en, id } from './i18n/dict'

export const LangCtx = createContext<{ lang: Lang; setLang: (l: Lang) => void; t: (k: string) => string }>({
  lang: 'en',
  setLang: () => {},
  t: (k: string) => en[k] ?? k,
})

export const useLang = () => useContext(LangCtx)

export function translate(lang: Lang, k: string): string {
  if (lang === 'id') return id[k] ?? en[k] ?? k
  if (lang === 'ar') return ar[k] ?? en[k] ?? k
  return en[k] ?? k
}
