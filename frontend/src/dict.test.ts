import { describe, expect, it } from 'vitest'
import { en, id } from './i18n/dict'

describe('i18n dict', () => {
  it('translates core nav keys in en + id', () => {
    expect(en['nav.dashboard']).toBe('Dashboard')
    expect(id['nav.dashboard']).toBe('Dasbor')
    expect(id['billing.invoices']).toBe('Faktur')
  })
  it('has no empty translations', () => {
    for (const [k, v] of Object.entries({ ...en, ...id })) {
      expect(v, k).not.toBe('')
    }
  })
})
