// Sinh PNG icons từ SVG (dùng chromium — không cần cài converter).
// Chạy: bun run scripts/gen-icons.mjs
// Output: <repo>/frontend/public + <repo>/dashboard/public
import { chromium } from '@playwright/test'
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const srcDir = join(root, 'public') // frontend/public (đã copy SVG)
const targets = [
  { svg: 'favicon.svg', out: 'favicon-16.png', w: 16, h: 16 },
  { svg: 'favicon.svg', out: 'favicon-32.png', w: 32, h: 32 },
  { svg: 'apple-touch-icon.svg', out: 'apple-touch-icon.png', w: 180, h: 180 },
  { svg: 'favicon-192.svg', out: 'icon-192.png', w: 192, h: 192 },
  { svg: 'favicon-512.svg', out: 'icon-512.png', w: 512, h: 512 },
  { svg: 'og-image.svg', out: 'og-image.png', w: 1200, h: 630 },
]

const browser = await chromium.launch()
const page = await browser.newPage()

// frontend/public = <root>/public ; dashboard/public = <root>/../dashboard/public
const dests = [join(root, 'public'), join(root, '..', 'dashboard', 'public')]
for (const outDir of dests) {
  mkdirSync(outDir, { recursive: true })
  for (const t of targets) {
    const svgPath = `file://${join(srcDir, t.svg)}`
    await page.setViewportSize({ width: t.w, height: t.h })
    await page.goto(svgPath, { waitUntil: 'load' })
    await page.waitForTimeout(100)
    const buf = await page.screenshot()
    const out = join(outDir, t.out)
    writeFileSync(out, buf)
    console.log('✓', out, `${buf.length} bytes`)
  }
}

await browser.close()
