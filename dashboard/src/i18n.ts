// i18n dashboard (vue-i18n) — locale vi/en, lưu lựa chọn vào localStorage.
// Backend trả msg theo Accept-Language (xem client.ts) — đồng bộ 1 nguồn ngôn ngữ.
import { createI18n } from 'vue-i18n'
import vi from './locales/vi'
import en from './locales/en'

export const LOCALE_KEY = 'gvs_locale'

export function getInitialLocale(): 'vi' | 'en' {
  const saved = localStorage.getItem(LOCALE_KEY)
  if (saved === 'vi' || saved === 'en') return saved
  return 'vi' // mặc định tiếng Việt (zero-config, khớp backend default)
}

export function setLocale(locale: 'vi' | 'en') {
  localStorage.setItem(LOCALE_KEY, locale)
  i18n.global.locale.value = locale
  document.documentElement.lang = locale
}

const i18n = createI18n({
  legacy: false, // composition API ($t / useI18n)
  locale: getInitialLocale(),
  fallbackLocale: 'vi',
  messages: { vi, en },
})

export default i18n
