import { describe, expect, it } from 'vitest'
import { ar, en, dirOf } from './i18n/dict'
import { translate } from './lang'

describe('arabic + rtl', () => {
  it('translates core nav keys', () => {
    expect(ar['nav.dashboard']).toBe('لوحة القيادة')
    expect(ar['billing.invoices']).toBe('الفواتير')
  })
  it('dirOf maps ar to rtl, others to ltr', () => {
    expect(dirOf('ar')).toBe('rtl')
    expect(dirOf('en')).toBe('ltr')
    expect(dirOf('id')).toBe('ltr')
  })
  it('translate falls back to english for uncovered keys', () => {
    expect(translate('ar', 'lab.hint')).toBe(en['lab.hint'])
    expect(translate('ar', 'nav.noc')).toBe(ar['nav.noc'])
  })
})
