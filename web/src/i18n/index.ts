import { createI18n } from 'vue-i18n'
import en from './locales/en'
import de from './locales/de'

export type SupportedLocale = 'en' | 'de'

const savedLocale = localStorage.getItem('locale') as SupportedLocale | null
const browserLocale = navigator.language.split('-')[0] as SupportedLocale
const defaultLocale: SupportedLocale = savedLocale || (browserLocale === 'de' ? 'de' : 'en')

const i18n = createI18n({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale: 'en',
  messages: {
    en,
    de
  }
})

export function setLocale(locale: SupportedLocale) {
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
  document.documentElement.setAttribute('lang', locale)
}

export function getLocale(): SupportedLocale {
  return i18n.global.locale.value as SupportedLocale
}

export default i18n
