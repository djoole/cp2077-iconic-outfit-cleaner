<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Check, ChevronDown, ChevronRight, FolderOpen, RefreshCw, Search, ShieldCheck, Sparkles } from 'lucide-vue-next'
import { DetectPaths, GeneratePatch, OpenPatchFolder, PickOutputRoot, PickScanRoot, Scan } from '../wailsjs/go/main/App'

type Item = { recordId: string; sourceFile: string; originalQuality: string; template: boolean }
type Mod = { id: string; name: string; sourceLabel: string; itemCount: number; fileCount: number; items: Item[] }
type ScanResult = { scanRoot: string; outputRoot: string; mods: Mod[]; totalItems: number; scannedFiles: number; tweaksFolders: number; usingVortex: boolean; patchPath: string; warnings: string[] }

const scanRoot = ref('')
const outputRoot = ref('')
const scan = ref<ScanResult | null>(null)
const selected = ref(new Set<string>())
const expanded = ref(new Set<string>())
const query = ref('')
const busy = ref(false)
const message = ref('')
const error = ref('')

const visibleMods = computed(() => {
  const needle = query.value.trim().toLocaleLowerCase()
  if (!scan.value || !needle) return scan.value?.mods ?? []
  return scan.value.mods.filter(mod =>
    mod.name.toLocaleLowerCase().includes(needle) ||
    mod.sourceLabel.toLocaleLowerCase().includes(needle) ||
    mod.items.some(item => item.recordId.toLocaleLowerCase().includes(needle))
  )
})
const selectedMods = computed(() => scan.value?.mods.filter(mod => selected.value.has(mod.id)) ?? [])
const selectedItems = computed(() => selectedMods.value.reduce((sum, mod) => sum + mod.itemCount, 0))

function replaceSet(target: typeof selected, values: Iterable<string>) {
  target.value = new Set(values)
}
function toggleSelected(id: string) {
  const next = new Set(selected.value)
  next.has(id) ? next.delete(id) : next.add(id)
  selected.value = next
}
function toggleExpanded(id: string) {
  const next = new Set(expanded.value)
  next.has(id) ? next.delete(id) : next.add(id)
  expanded.value = next
}
function selectVisible(value: boolean) {
  const next = new Set(selected.value)
  for (const mod of visibleMods.value) value ? next.add(mod.id) : next.delete(mod.id)
  selected.value = next
}

function invalidateScan() {
  scan.value = null
  selected.value = new Set()
  expanded.value = new Set()
  message.value = ''
}
async function chooseScanRoot() {
  const path = await PickScanRoot(scanRoot.value)
  if (path && path !== scanRoot.value) {
    scanRoot.value = path
    invalidateScan()
  }
}
async function chooseOutputRoot() {
  const path = await PickOutputRoot(outputRoot.value)
  if (path && path !== outputRoot.value) {
    outputRoot.value = path
    invalidateScan()
  }
}
async function runScan() {
  busy.value = true; error.value = ''; message.value = ''
  try {
    const result = await Scan(scanRoot.value, outputRoot.value) as ScanResult
    scan.value = result
    scanRoot.value = result.scanRoot
    outputRoot.value = result.outputRoot
    replaceSet(selected, result.mods.map(mod => mod.id))
    expanded.value = new Set()
    message.value = result.mods.length
      ? `${result.mods.length} affected mods found — all selected by default.`
      : 'No modded Iconic clothing records were found.'
  } catch (e) { error.value = String(e) }
  finally { busy.value = false }
}
async function generate() {
  busy.value = true; error.value = ''; message.value = ''
  try {
    const result = await GeneratePatch([...selected.value])
    message.value = `Patch generated: ${result.patchSummary}.`
  } catch (e) { error.value = String(e) }
  finally { busy.value = false }
}
async function openPatch() {
  error.value = ''
  try { await OpenPatchFolder() }
  catch (e) { error.value = String(e) }
}

onMounted(async () => {
  if (import.meta.env.DEV && new URLSearchParams(location.search).has('demo')) {
    scanRoot.value = 'C:\\Tools\\MO2\\mods'
    outputRoot.value = 'C:\\Tools\\MO2\\mods\\Iconic Outfit Cleaner'
    scan.value = {
      scanRoot: scanRoot.value,
      outputRoot: outputRoot.value,
      totalItems: 2072,
      scannedFiles: 1941,
      tweaksFolders: 286,
      usingVortex: true,
      patchPath: outputRoot.value + '\\r6\\scripts\\IconicOutfitCleaner\\generated.reds',
      warnings: [],
      mods: [
        { id: 'adshield-tactical', name: 'Adshield Tactical Crop Top - Angel', sourceLabel: 'Adshield Tactical Crop Top - Angel-8406-1-01-1718072348', itemCount: 42, fileCount: 1, items: [{ recordId: 'Items.Adshield_Tac_Crop_Top_Black_FL', sourceFile: 'r6/tweaks/adshield/Tac_Crop_Top.yaml', originalQuality: 'Quality.Legendary', template: false }] },
        { id: 'axellysse-angela', name: 'Axellysse Angela Top', sourceLabel: 'Axellysse Angela Top-20114-2-0', itemCount: 12, fileCount: 1, items: [{ recordId: 'Items.axellysse_angela_top_$(base_color)', sourceFile: 'r6/tweaks/axellysse/axellysse_angela_top.yaml', originalQuality: 'Quality.Legendary', template: true }] },
        { id: 'hyst-outfit', name: 'Hyst Casual Day Outfit', sourceLabel: 'Hyst Casual Day Outfit-9934-1-3', itemCount: 18, fileCount: 2, items: [{ recordId: 'Items.hyst_casual_day_outfit_black', sourceFile: 'r6/tweaks/hyst_casual_day_outfit.yaml', originalQuality: 'Quality.Legendary', template: false }] },
      ],
    }
    replaceSet(selected, scan.value.mods.map(mod => mod.id))
    message.value = '240 affected mods found — all selected by default.'
    return
  }
  const defaults = await DetectPaths()
  scanRoot.value = defaults.scanRoot
  outputRoot.value = defaults.outputRoot
  if (scanRoot.value && outputRoot.value) await runScan()
})
</script>

<template>
  <main>
    <header class="hero">
      <div class="brand-mark"><Sparkles :size="28" /></div>
      <div>
        <p class="eyebrow">TWEAKXL COMPATIBILITY PATCHER</p>
        <h1>CP2077 Iconic Outfit Cleaner</h1>
        <p class="subtitle">Remove iconic status from selected modded outfits.</p>
      </div>
      <div v-if="scan" class="scan-badge" :title="scan.usingVortex ? 'Exact file ownership was read from vortex.deployment.json.' : 'Source names were inferred from discovered mod folders.'"><ShieldCheck :size="17" />{{ scan.usingVortex ? 'Vortex source mapping active' : 'Folder source mapping active' }}</div>
    </header>

    <section class="path-card">
      <div class="path-field">
        <label>Mod scan root <span>Searches recursively for every r6\tweaks folder</span></label>
        <div class="path-row">
          <input v-model="scanRoot" placeholder="…\Cyberpunk 2077 or …\MO2\mods" @input="invalidateScan" @keyup.enter="runScan" />
          <button class="icon-button" title="Browse for scan folder" @click="chooseScanRoot"><FolderOpen :size="19" /></button>
        </div>
      </div>
      <div class="path-field">
        <label>Patch destination <span>The generated r6\scripts folder will be created here</span></label>
        <div class="path-row">
          <input v-model="outputRoot" placeholder="…\Cyberpunk 2077 or …\MO2\mods\Iconic Outfit Cleaner" @input="invalidateScan" @keyup.enter="runScan" />
          <button class="icon-button" title="Browse for patch destination" @click="chooseOutputRoot"><FolderOpen :size="19" /></button>
          <button class="primary compact" :disabled="busy || !scanRoot || !outputRoot" @click="runScan"><RefreshCw :size="17" :class="{ spinning: busy }" />Scan</button>
        </div>
      </div>
    </section>

    <div v-if="error" class="notice error">{{ error }}</div>
    <div v-else-if="message" class="notice success"><Check :size="17" />{{ message }}</div>

    <section v-if="scan" class="workspace">
      <div class="toolbar">
        <div class="metrics">
          <div><strong>{{ scan.mods.length }}</strong><span>affected mods</span></div>
          <div><strong>{{ scan.totalItems }}</strong><span>records Iconic</span></div>
          <div><strong>{{ scan.scannedFiles }}</strong><span>YAML files scanned</span></div>
          <div><strong>{{ scan.tweaksFolders }}</strong><span>tweak folders found</span></div>
        </div>
        <div class="search"><Search :size="17" /><input v-model="query" placeholder="Filter mods or Item IDs…" /></div>
      </div>

      <div class="selection-bar">
        <span><strong>{{ selected.size }}</strong> / {{ scan.mods.length }} mods selected · <strong>{{ selectedItems }}</strong> records</span>
        <div><button class="text-button" @click="selectVisible(true)">Select all</button><button class="text-button" @click="selectVisible(false)">Select none</button></div>
      </div>

      <div class="mod-list">
        <article v-for="mod in visibleMods" :key="mod.id" class="mod-card" :class="{ excluded: !selected.has(mod.id) }">
          <button class="expand-button" @click="toggleExpanded(mod.id)">
            <ChevronDown v-if="expanded.has(mod.id)" :size="18" /><ChevronRight v-else :size="18" />
          </button>
          <label class="checkbox-wrap">
            <input type="checkbox" :checked="selected.has(mod.id)" @change="toggleSelected(mod.id)" />
            <span class="fake-checkbox"><Check :size="14" /></span>
          </label>
          <div class="mod-info" @click="toggleSelected(mod.id)">
            <h2>{{ mod.name }}</h2>
            <p>{{ mod.sourceLabel }}</p>
          </div>
          <div class="counts"><span>{{ mod.itemCount }} records</span><span>{{ mod.fileCount }} {{ mod.fileCount > 1 ? 'files' : 'file' }}</span></div>
          <div v-if="expanded.has(mod.id)" class="items">
            <div v-for="item in mod.items" :key="item.recordId" class="item-row">
              <code>{{ item.recordId }}</code>
              <span v-if="item.template" class="tag">template</span>
              <span class="quality">{{ item.originalQuality || 'Iconic modifier' }}</span>
              <small>{{ item.sourceFile }}</small>
            </div>
          </div>
        </article>
        <div v-if="visibleMods.length === 0" class="empty">No mods match this filter.</div>
      </div>

      <footer class="action-bar">
        <div><span>TweakXL script output</span><code>{{ scan.patchPath }}</code></div>
        <button class="secondary" @click="openPatch"><FolderOpen :size="17" />Open</button>
        <button class="primary" :disabled="busy || selected.size === 0" @click="generate"><Sparkles :size="18" />Generate patch</button>
      </footer>
    </section>
  </main>
</template>
