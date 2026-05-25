import { cp, mkdir, rm, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const src = path.resolve(__dirname, '..', 'dist')
const dest = path.resolve(__dirname, '..', '..', 'internal', 'embed', 'dist')

await rm(dest, { recursive: true, force: true })
await mkdir(dest, { recursive: true })
await cp(src, dest, { recursive: true })
await writeFile(path.join(dest, '.gitkeep'), '')
console.log(`copied ${src} → ${dest}`)
