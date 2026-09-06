<template>
  <div class="cc-root">
    <div class="bg-blobs"><div class="blob blob-1"></div><div class="blob blob-2"></div><div class="blob blob-3"></div></div>
    <div class="grain"></div>
    <div class="spotlight" ref="spotlightEl"></div>

    <div class="ticker-bar" v-if="tickerItems.length">
      <div class="ticker-track">
        <span v-for="(t, i) in tickerItems.concat(tickerItems)" :key="i">{{ t }}</span>
      </div>
    </div>

    <div class="shell">
      <div class="header-bar">
        <div>
          <h1 class="header-title">Command Center</h1>
          <div class="header-sub">Real-time insight into all app activity</div>
        </div>
        <div class="refresh-row">
          <span class="refresh-dot"></span>
          {{ lastUpdatedLabel }}
          <button class="refresh-btn" type="button" @click="refresh" :disabled="analyticsLoading">
            <v-icon icon="mdi-refresh" size="14" />
          </button>
        </div>
      </div>

      <div v-if="analyticsError" class="error-banner">{{ analyticsError }}</div>

      <div class="stats-row">
        <div class="glow-card stat-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(45,212,240,0.14);color:var(--cc-cyan)">
              <v-icon icon="mdi-account-group" size="16" />
            </div>
            <span class="stat-value" :data-count="overview?.totalUsers ?? 0">0</span>
            <span class="stat-label">Total users</span>
            <div class="stat-breakdown">
              <span><b>{{ overview?.usersByRole?.doctor ?? 0 }}</b> doctors</span>
              <span><b>{{ overview?.usersByRole?.admin ?? 0 }}</b> admins</span>
              <span><b>{{ overview?.usersByRole?.superuser ?? 0 }}</b> superusers</span>
              <span><b>{{ overview?.usersByRole?.user ?? 0 }}</b> demo</span>
            </div>
          </div>
        </div>

        <div class="glow-card stat-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(91,157,245,0.14);color:var(--cc-blue)">
              <v-icon icon="mdi-file-document-multiple" size="16" />
            </div>
            <span class="stat-value" :data-count="overview?.totalSessions ?? 0">0</span>
            <span class="stat-label">Total sessions</span>
            <div class="stat-breakdown">
              <b>{{ overview?.sessionsLast7d ?? 0 }}</b><span>this week</span>
              <span v-if="weeklyTrendPct !== null" class="stat-trend" :class="{ down: weeklyTrendPct < 0 }">
                <v-icon :icon="weeklyTrendPct < 0 ? 'mdi-arrow-down' : 'mdi-arrow-up'" size="11" />
                {{ Math.abs(weeklyTrendPct) }}%
              </span>
            </div>
          </div>
        </div>

        <div class="glow-card stat-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(168,85,247,0.14);color:var(--cc-violet)">
              <v-icon icon="mdi-calendar-week" size="16" />
            </div>
            <span class="stat-value" :data-count="overview?.sessionsLast7d ?? 0">0</span>
            <span class="stat-label">Sessions this week</span>
            <div class="stat-mini-bars">
              <span v-for="(v, i) in last7DayTotals" :key="i"
                    :style="{ height: barHeight(v) + 'px', background: 'var(--cc-cyan)', animationDelay: (1.0 + i * 0.05) + 's' }"></span>
            </div>
          </div>
        </div>

        <div class="glow-card stat-card tilt-card" :class="{ crit: (overview?.pendingApprovals ?? 0) > 0 }" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(242,97,122,0.14);color:var(--cc-crit)">
              <v-icon icon="mdi-clock-alert-outline" size="16" />
            </div>
            <span class="stat-value" :class="{ crit: (overview?.pendingApprovals ?? 0) > 0 }" :data-count="overview?.pendingApprovals ?? 0">0</span>
            <span class="stat-label">Pending approvals</span>
            <router-link v-if="(overview?.pendingApprovals ?? 0) > 0" to="/pending-users" class="stat-warn-note">Needs review →</router-link>
            <span v-else class="stat-warn-note" style="color:var(--cc-good)">All clear</span>
          </div>
        </div>
      </div>

      <div class="mid-grid">
        <div class="glow-card chart-panel tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body">
            <div class="panel-head"><h2 class="panel-title">30-day usage</h2><span class="panel-sub">{{ chartRangeLabel }}</span></div>
            <div class="legend">
              <span class="legend-item" :class="{ off: hiddenSeries.ambient }" @click="toggleSeries('ambient')"><span class="legend-dot" style="background:var(--cc-cyan)"></span>Ambient</span>
              <span class="legend-item" :class="{ off: hiddenSeries.file }" @click="toggleSeries('file')"><span class="legend-dot" style="background:var(--cc-blue)"></span>File transcription</span>
              <span class="legend-item" :class="{ off: hiddenSeries.dictation }" @click="toggleSeries('dictation')"><span class="legend-dot" style="background:var(--cc-violet)"></span>Dictation</span>
            </div>
            <div class="chart-wrap">
              <svg ref="chartSvg" viewBox="0 0 640 216" preserveAspectRatio="none"></svg>
              <div class="chart-tooltip" ref="chartTooltip"></div>
            </div>
          </div>
        </div>

        <div class="glow-card sidebar-panel tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body sidebar-body">
            <div class="panel-head"><h2 class="panel-title">Quick stats</h2></div>
            <div class="side-row"><span class="side-label">Sessions (24h)</span><span class="side-value">{{ overview?.sessionsLast24h ?? 0 }}</span></div>
            <div class="side-row"><span class="side-label">Sessions (30d)</span><span class="side-value">{{ overview?.sessionsLast30d ?? 0 }}</span></div>
            <div class="side-row"><span class="side-label">Demo conversion rate</span><span class="side-value">{{ demoConversionRate }}%</span></div>
            <div class="side-row"><span class="side-label">Demo accounts at limit</span><span class="side-value" :class="{ good: (overview?.demoAccountsAtLimit ?? 0) === 0 }">{{ overview?.demoAccountsAtLimit ?? 0 }}</span></div>
            <div class="side-row"><span class="side-label">Total demo accounts</span><span class="side-value">{{ overview?.demoAccountsCount ?? 0 }}</span></div>
          </div>
        </div>
      </div>

      <div class="glow-card funnel-panel tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
        <div class="glow-inner panel-body">
          <div class="panel-head">
            <h2 class="panel-title">Trial account funnel</h2>
            <span class="panel-sub">Sorted by closest to {{ DEMO_TRIAL_LIMIT }}-use limit</span>
          </div>
          <div v-if="sortedFunnel.length === 0" class="empty-note">No demo accounts yet.</div>
          <div class="table-scroll" v-else>
            <table class="funnel">
              <thead>
                <tr><th>Username</th><th>Requested role</th><th>Status</th><th>Core feature usage</th><th>AI features tried</th></tr>
              </thead>
              <tbody>
                <tr v-for="(u, i) in sortedFunnel" :key="u.userId" class="demo-row"
                    :class="{ 'at-limit': u.atLimit, 'near-limit': !u.atLimit && u.nearLimit }"
                    :style="{ animationDelay: (0.06 * i) + 's' }"
                    @click="goToUser(u.userId)">
                  <td><div class="u-name">{{ u.username }}</div><div class="u-name-sub">Requested {{ relativeTime(u.createdAt) }}</div></td>
                  <td><span class="chip chip-role">{{ formatRole(u.requestedRole) }}</span></td>
                  <td><span class="chip" :class="'chip-' + u.approvalStatus">{{ capitalize(u.approvalStatus) }}</span></td>
                  <td>
                    <div class="usage-cell">
                      <div class="u-bar"><div class="u-bar-label">AMB</div><div class="u-bar-track"><div class="u-bar-fill" :style="{ width: pct(u.featureUsage.ambient) + '%', background: 'var(--cc-cyan)' }"></div></div><div class="u-bar-num">{{ u.featureUsage.ambient || 0 }}/{{ DEMO_TRIAL_LIMIT }}</div></div>
                      <div class="u-bar"><div class="u-bar-label">FILE</div><div class="u-bar-track"><div class="u-bar-fill" :style="{ width: pct(u.featureUsage.file_transcription) + '%', background: 'var(--cc-blue)' }"></div></div><div class="u-bar-num">{{ u.featureUsage.file_transcription || 0 }}/{{ DEMO_TRIAL_LIMIT }}</div></div>
                      <div class="u-bar"><div class="u-bar-label">DICT</div><div class="u-bar-track"><div class="u-bar-fill" :style="{ width: pct(u.featureUsage.dictation) + '%', background: 'var(--cc-violet)' }"></div></div><div class="u-bar-num">{{ u.featureUsage.dictation || 0 }}/{{ DEMO_TRIAL_LIMIT }}</div></div>
                    </div>
                  </td>
                  <td>
                    <div class="ai-dot-group-label">SUM · DOC · SRCH</div>
                    <div class="ai-dots">
                      <span class="ai-dot" :class="{ on: (u.featureUsage.ai_summary || 0) > 0 }"></span>
                      <span class="ai-dot" :class="{ on: (u.featureUsage.document_generation || 0) > 0 }"></span>
                      <span class="ai-dot" :class="{ on: (u.featureUsage.search || 0) > 0 }"></span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="glow-card activity-panel tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
        <div class="glow-inner panel-body">
          <div class="panel-head">
            <h2 class="panel-title">Recent activity across all users</h2>
            <span class="panel-sub">Latest {{ recentSessionsAllUsers.length }}{{ overview ? ' of ' + overview.totalSessions : '' }}</span>
          </div>
          <div v-if="recentSessionsAllUsers.length === 0" class="empty-note">No sessions yet.</div>
          <div v-else>
            <div v-for="(s, i) in recentSessionsAllUsers" :key="s.id" class="activity-item"
                 :style="{ animationDelay: (0.06 * i) + 's' }" @click="goToUser(s.userId)">
              <span class="user-avatar" :style="s.isDemo ? { background: 'linear-gradient(145deg,var(--cc-warn),var(--cc-crit) 130%)' } : {}">{{ initials(s.userName || s.username) }}</span>
              <div class="activity-main">
                <div class="activity-user-row">
                  <span class="activity-user">{{ s.userName }}<span v-if="s.isDemo" class="demo-tag"> · demo</span></span>
                  <span class="type-chip" :class="typeClass(s.type)">{{ typeLabel(s.type) }}</span>
                </div>
                <div class="activity-title">{{ s.title }}</div>
                <div class="activity-snippet">"...{{ s.transcript }}..."</div>
              </div>
              <div class="activity-right"><span class="activity-time">{{ relativeTime(s.createdAt) }}</span><span v-if="s.document" class="doc-chip">✓ Document</span></div>
            </div>
          </div>
          <router-link to="/users" class="view-all">View all users →</router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import {
  loadAnalytics, analyticsOverview, usageTimeseries, demoFunnelUsers,
  recentSessionsAllUsers, analyticsLoading, analyticsError, DEMO_TRIAL_LIMIT
} from '@/stores/adminAnalytics'

const router = useRouter()
const overview = analyticsOverview
const lastUpdated = ref<Date | null>(null)

const spotlightEl = ref<HTMLElement | null>(null)
const chartSvg = ref<SVGSVGElement | null>(null)
const chartTooltip = ref<HTMLElement | null>(null)
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

async function refresh() {
  await loadAnalytics(30)
  lastUpdated.value = new Date()
  await nextTick()
  animateCounts()
  animateMiniBars()
  drawChart()
}

onMounted(refresh)

const lastUpdatedLabel = computed(() => {
  if (!lastUpdated.value) return 'Loading…'
  const secs = Math.round((Date.now() - lastUpdated.value.getTime()) / 1000)
  if (secs < 5) return 'Updated just now'
  if (secs < 60) return `Updated ${secs}s ago`
  return `Updated ${Math.round(secs / 60)}m ago`
})

// --- Ticker: real, honestly-derived strings only ---
const tickerItems = computed(() => {
  if (!overview.value) return []
  const items: string[] = []
  items.push(`${overview.value.sessionsLast24h} sessions in the last 24 hours`)
  if (overview.value.pendingApprovals > 0) items.push(`${overview.value.pendingApprovals} pending approval${overview.value.pendingApprovals === 1 ? '' : 's'} need review`)
  items.push(`${demoConversionRate.value}% demo conversion rate`)
  if (overview.value.demoAccountsAtLimit > 0) items.push(`${overview.value.demoAccountsAtLimit} demo account${overview.value.demoAccountsAtLimit === 1 ? '' : 's'} at the trial limit`)
  items.push(`${overview.value.totalUsers} total users`)
  return items
})

// --- Weekly trend — both sides use the same backend-computed rolling
// 7-day windows as the "Total sessions" stat card's own sessionsLast7d, so
// the two numbers never disagree the way comparing against the 30-day
// timeseries' calendar-day buckets could (different window definitions).
const weeklyTrendPct = computed<number | null>(() => {
  const o = overview.value
  if (!o || o.sessionsPrev7d === 0) return null
  return Math.round(((o.sessionsLast7d - o.sessionsPrev7d) / o.sessionsPrev7d) * 100)
})

const last7DayTotals = computed(() => usageTimeseries.value.slice(-7).map(d => d.ambient + d.file + d.dictation))
function barHeight(v: number) {
  const max = Math.max(1, ...last7DayTotals.value)
  return Math.max(3, Math.round((v / max) * 22))
}

const demoConversionRate = computed(() => {
  const total = demoFunnelUsers.value.length
  if (total === 0) return 0
  const approved = demoFunnelUsers.value.filter(u => u.approvalStatus === 'approved').length
  return Math.round((approved / total) * 100)
})

const chartRangeLabel = computed(() => {
  const s = usageTimeseries.value
  if (s.length === 0) return ''
  const fmt = (d: string) => new Date(d + 'T00:00:00').toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  return `${fmt(s[0].date)} – ${fmt(s[s.length - 1].date)}`
})

// --- Demo funnel, sorted by closest-to-limit ---
const sortedFunnel = computed(() => {
  return demoFunnelUsers.value.map(u => {
    const amb = u.featureUsage.ambient || 0
    const file = u.featureUsage.file_transcription || 0
    const dict = u.featureUsage.dictation || 0
    const coreMax = Math.max(amb, file, dict)
    const total = amb + file + dict
    return { ...u, coreMax, total, atLimit: coreMax >= DEMO_TRIAL_LIMIT, nearLimit: coreMax >= DEMO_TRIAL_LIMIT - 1 }
  }).sort((a, b) => b.coreMax - a.coreMax || b.total - a.total)
})

function pct(v: number | undefined) {
  return Math.min(100, Math.round(((v || 0) / DEMO_TRIAL_LIMIT) * 100))
}

function goToUser(userId: string) {
  router.push(`/user-dashboard/${userId}`)
}

// --- formatting helpers ---
function capitalize(s: string) { return s ? s.charAt(0).toUpperCase() + s.slice(1) : s }
function formatRole(role: string) {
  if (role === 'superuser') return 'Super User'
  if (role === 'doctor') return 'Doctor'
  return role ? capitalize(role) : 'User'
}
function initials(name: string) {
  if (!name) return '?'
  const parts = name.replace(/^Dr\.\s*/i, '').trim().split(/\s+/)
  return parts.slice(0, 2).map(p => p[0]?.toUpperCase() || '').join('')
}
function typeClass(type: string) {
  if (type === 'ambient') return 'type-ambient'
  if (type === 'file-transcription') return 'type-file'
  if (type === 'dictation') return 'type-dictation'
  return 'type-file'
}
function typeLabel(type: string) {
  if (type === 'ambient') return 'Ambient'
  if (type === 'file-transcription') return 'File'
  if (type === 'dictation') return 'Dictation'
  return type
}
function relativeTime(iso: string) {
  const then = new Date(iso).getTime()
  const secs = Math.round((Date.now() - then) / 1000)
  if (secs < 60) return 'just now'
  const mins = Math.round(secs / 60)
  if (mins < 60) return `${mins}m ago`
  const hours = Math.round(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  if (days === 1) return 'yesterday'
  if (days < 30) return `${days} days ago`
  const months = Math.round(days / 30)
  return `${months} month${months === 1 ? '' : 's'} ago`
}

// --- cursor spotlight + tilt (skipped under reduced-motion) ---
function onTilt(e: MouseEvent) {
  if (reducedMotion) return
  const card = e.currentTarget as HTMLElement
  const r = card.getBoundingClientRect()
  const px = (e.clientX - r.left) / r.width - 0.5
  const py = (e.clientY - r.top) / r.height - 0.5
  card.style.transform = `perspective(700px) rotateX(${-py * 5}deg) rotateY(${px * 6}deg) translateZ(2px)`
}
function onTiltLeave(e: MouseEvent) {
  (e.currentTarget as HTMLElement).style.transform = ''
}
function onSpotlightMove(e: MouseEvent) {
  spotlightEl.value?.style.setProperty('--sx', e.clientX + 'px')
  spotlightEl.value?.style.setProperty('--sy', e.clientY + 'px')
}
if (typeof window !== 'undefined' && !reducedMotion) {
  window.addEventListener('mousemove', onSpotlightMove)
  onUnmounted(() => window.removeEventListener('mousemove', onSpotlightMove))
}

// --- count-up numbers ---
function animateCounts() {
  const els = document.querySelectorAll<HTMLElement>('.cc-root .stat-value[data-count]')
  els.forEach((el, i) => {
    const target = parseInt(el.dataset.count || '0', 10)
    setTimeout(() => countUp(el, target, 1200), 200 + i * 120)
  })
}
function countUp(el: HTMLElement, target: number, duration: number) {
  if (reducedMotion) { el.textContent = String(target); return }
  let start: number | null = null
  function step(ts: number) {
    if (start === null) start = ts
    const p = Math.min((ts - start) / duration, 1)
    const eased = p < 1 ? 1 - Math.pow(2, -10 * p) * Math.cos((p * 10 - 0.75) * (2 * Math.PI) / 3) : 1
    el.textContent = String(Math.max(0, Math.round(eased * target)))
    if (p < 1) requestAnimationFrame(step)
    else {
      el.textContent = String(target)
      el.classList.add('done')
      setTimeout(() => el.classList.remove('done'), 900)
    }
  }
  requestAnimationFrame(step)
}
function animateMiniBars() {
  document.querySelectorAll<HTMLElement>('.cc-root .stat-mini-bars span').forEach(el => {
    el.style.opacity = '0'
    el.style.transform = 'scaleY(0)'
    requestAnimationFrame(() => {
      el.style.transition = 'opacity 0.5s cubic-bezier(.2,1.4,.4,1), transform 0.5s cubic-bezier(.2,1.4,.4,1)'
      el.style.opacity = '1'
      el.style.transform = 'scaleY(1)'
    })
  })
}

// --- hand-drawn SVG usage chart (no charting library — matches design mockup) ---
const hiddenSeries = reactive<Record<string, boolean>>({ ambient: false, file: false, dictation: false })
function toggleSeries(key: string) {
  hiddenSeries[key] = !hiddenSeries[key]
  const svg = chartSvg.value
  if (!svg) return
  const g = svg.querySelector(`g[data-series="${key}"]`) as SVGGElement | null
  if (g) g.style.display = hiddenSeries[key] ? 'none' : ''
}

// The comet dot on each series animates on an endless requestAnimationFrame
// loop. chartGeneration is bumped on every drawChart() call (refresh, or a
// remount) and checked inside that loop — a stale loop from a previous
// draw (or a redraw after unmount) sees its generation no longer matches
// and simply stops rescheduling itself, instead of running forever in the
// background accumulating one orphaned loop per refresh.
let chartGeneration = 0
onUnmounted(() => { chartGeneration = -1 })

function drawChart() {
  const myGen = ++chartGeneration
  const svg = chartSvg.value
  if (!svg) return
  svg.innerHTML = ''
  const data = usageTimeseries.value
  if (data.length === 0) return

  const W = 640, H = 216, padL = 30, padR = 8, padT = 10, padB = 22
  const days = data.length
  const maxV = Math.max(5, Math.ceil(Math.max(...data.map(d => Math.max(d.ambient, d.file, d.dictation))) / 5) * 5)
  const x = (i: number) => padL + (i / (days - 1)) * (W - padL - padR)
  const y = (v: number) => H - padB - (v / maxV) * (H - padT - padB)

  const ns = 'http://www.w3.org/2000/svg'
  function el(tag: string, attrs: Record<string, any>) {
    const e = document.createElementNS(ns, tag)
    for (const k in attrs) e.setAttribute(k, String(attrs[k]))
    return e
  }
  function catmullRom(pts: [number, number][]) {
    if (pts.length < 3) return 'M' + pts.map(p => p[0] + ',' + p[1]).join('L')
    let d = 'M' + pts[0][0] + ',' + pts[0][1] + ' '
    for (let i = 0; i < pts.length - 1; i++) {
      const p0 = pts[i === 0 ? 0 : i - 1], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2 < pts.length ? i + 2 : i + 1]
      const c1x = p1[0] + (p2[0] - p0[0]) / 6, c1y = p1[1] + (p2[1] - p0[1]) / 6
      const c2x = p2[0] - (p3[0] - p1[0]) / 6, c2y = p2[1] - (p3[1] - p1[1]) / 6
      d += 'C' + c1x + ',' + c1y + ' ' + c2x + ',' + c2y + ' ' + p2[0] + ',' + p2[1] + ' '
    }
    return d
  }

  const defs = el('defs', {})
  ;[['cyan', '#2dd4f0'], ['blue', '#5b9df5'], ['violet', '#a855f7']].forEach(([name, color]) => {
    const grad = el('linearGradient', { id: 'ccfill-' + name, x1: 0, y1: 0, x2: 0, y2: 1 })
    const s1 = el('stop', { offset: '0%', 'stop-color': color, 'stop-opacity': 0.32 })
    const s2 = el('stop', { offset: '100%', 'stop-color': color, 'stop-opacity': 0 })
    grad.appendChild(s1); grad.appendChild(s2); defs.appendChild(grad)
  })
  svg.appendChild(defs)

  const gridGroup = el('g', {})
  for (let gy = 0; gy <= 4; gy++) {
    const vy = padT + (gy / 4) * (H - padT - padB)
    gridGroup.appendChild(el('line', { x1: padL, y1: vy, x2: W - padR, y2: vy, style: 'stroke:rgba(var(--v-theme-on-surface),0.12)', 'stroke-width': 1 }))
    const val = Math.round(maxV - (gy / 4) * maxV)
    const t = el('text', { x: 4, y: vy + 3, 'font-size': 9, fill: '#6b6488', 'font-family': 'JetBrains Mono, monospace' })
    t.textContent = String(val)
    gridGroup.appendChild(t)
  }
  const labelEvery = Math.max(1, Math.round(days / 6))
  for (let gi = 0; gi < days; gi += labelEvery) {
    const t2 = el('text', { x: x(gi), y: H - 5, 'font-size': 9, fill: '#6b6488', 'font-family': 'JetBrains Mono, monospace', 'text-anchor': 'middle' })
    const d = new Date(data[gi].date + 'T00:00:00')
    t2.textContent = `${d.getMonth() + 1}/${d.getDate()}`
    gridGroup.appendChild(t2)
  }
  svg.appendChild(gridGroup)

  const seriesMeta: Array<{ key: 'ambient' | 'file' | 'dictation', color: string, fill: string }> = [
    { key: 'ambient', color: '#2dd4f0', fill: 'url(#ccfill-cyan)' },
    { key: 'file', color: '#5b9df5', fill: 'url(#ccfill-blue)' },
    { key: 'dictation', color: '#a855f7', fill: 'url(#ccfill-violet)' }
  ]
  seriesMeta.forEach((m, si) => {
    const pts: [number, number][] = data.map((d, i) => [x(i), y(d[m.key])])
    const linePath = catmullRom(pts)
    const areaPath = linePath + ' L' + x(days - 1) + ',' + (H - padB) + ' L' + x(0) + ',' + (H - padB) + ' Z'
    const g = el('g', { 'data-series': m.key })
    if (hiddenSeries[m.key]) g.style.display = 'none'
    const area = el('path', { d: areaPath, fill: m.fill, opacity: 0 })
    const line = el('path', { d: linePath, fill: 'none', stroke: m.color, 'stroke-width': 2.4, 'stroke-linecap': 'round', style: `filter:drop-shadow(0 0 5px ${m.color})` })
    const comet = el('circle', { r: 3.4, fill: '#fff', style: `filter:drop-shadow(0 0 6px ${m.color})`, opacity: 0 })
    g.appendChild(area); g.appendChild(line)
    pts.forEach(p => g.appendChild(el('circle', { cx: p[0], cy: p[1], r: 2.2, fill: m.color, opacity: 0, class: 'chart-dot' })))
    g.appendChild(comet)
    svg.appendChild(g)

    if (!reducedMotion) {
      const len = (line as SVGPathElement).getTotalLength()
      ;(line as SVGPathElement).style.strokeDasharray = String(len)
      ;(line as SVGPathElement).style.strokeDashoffset = String(len)
      ;(line as SVGPathElement).style.transition = 'stroke-dashoffset 1.6s cubic-bezier(.4,0,.2,1)'
      setTimeout(() => {
        (line as SVGPathElement).style.strokeDashoffset = '0'
        setTimeout(() => { (area as SVGPathElement).style.transition = 'opacity 0.6s ease'; (area as SVGPathElement).style.opacity = '1' }, 1350)
        g.querySelectorAll('.chart-dot').forEach((dd, i) => {
          setTimeout(() => { (dd as SVGElement).style.transition = 'opacity 0.3s ease'; (dd as SVGElement).style.opacity = '1' }, i * (1600 / days))
        })
        setTimeout(() => animateComet(line as SVGPathElement, comet as SVGCircleElement, len), 1650)
      }, 600 + si * 140)
    } else {
      ;(area as SVGPathElement).style.opacity = '1'
      g.querySelectorAll('.chart-dot').forEach(dd => { (dd as SVGElement).style.opacity = '1' })
    }
  })

  function animateComet(line: SVGPathElement, comet: SVGCircleElement, len: number) {
    comet.style.transition = 'opacity 0.4s ease'
    comet.style.opacity = '0.9'
    const dur = 3800
    let t0: number | null = null
    function frame(ts: number) {
      // A newer drawChart() call (refresh) or unmount bumped/invalidated
      // chartGeneration — stop instead of looping forever on a detached SVG.
      if (myGen !== chartGeneration) return
      if (t0 === null) t0 = ts
      const p = ((ts - t0) % dur) / dur
      const pt = line.getPointAtLength(p * len)
      comet.setAttribute('cx', String(pt.x)); comet.setAttribute('cy', String(pt.y))
      requestAnimationFrame(frame)
    }
    requestAnimationFrame(frame)
  }

  const tooltip = chartTooltip.value
  if (tooltip) {
    svg.onmousemove = (e: MouseEvent) => {
      const rect = svg.getBoundingClientRect()
      const relX = (e.clientX - rect.left) / rect.width * W
      let idx = Math.round((relX - padL) / (W - padL - padR) * (days - 1))
      idx = Math.max(0, Math.min(days - 1, idx))
      const d = data[idx]
      const px = x(idx) / W * rect.width
      const py = y(Math.max(d.ambient, d.file, d.dictation)) / H * rect.height
      tooltip.style.left = px + 'px'; tooltip.style.top = (py - 10) + 'px'; tooltip.style.opacity = '1'
      const dt = new Date(d.date + 'T00:00:00')
      tooltip.innerHTML =
        `<div class="tt-date">${dt.toLocaleDateString()}</div>` +
        `<div class="tt-row"><span class="tt-dot" style="background:#2dd4f0"></span>Ambient<span class="tt-val">${d.ambient}</span></div>` +
        `<div class="tt-row"><span class="tt-dot" style="background:#5b9df5"></span>File<span class="tt-val">${d.file}</span></div>` +
        `<div class="tt-row"><span class="tt-dot" style="background:#a855f7"></span>Dictation<span class="tt-val">${d.dictation}</span></div>`
    }
    svg.onmouseleave = () => { tooltip.style.opacity = '0' }
  }
}
</script>

<style scoped>
.cc-root {
  --cc-cyan: #2dd4f0; --cc-blue: #5b9df5; --cc-violet: #a855f7; --cc-rose: #fb5aa0;
  --cc-good: #34e5a8; --cc-warn: #fbbf24; --cc-crit: #fb7185;
  --cc-crit-soft: rgba(251,113,133,0.14); --cc-warn-soft: rgba(251,191,36,0.14);
  --cc-panel: rgba(var(--v-theme-on-surface),0.045); --cc-border: rgba(var(--v-theme-on-surface),0.08);
  position: relative;
  font-variant-numeric: tabular-nums;
  font-family: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  color: rgb(var(--v-theme-on-background));
  margin: -16px; padding: 16px;
  overflow: hidden;
  border-radius: 12px;
}
.cc-root h1, .cc-root h2 { font-family: 'Space Grotesk', sans-serif; margin: 0; }
.cc-root .mono { font-family: 'JetBrains Mono', monospace; }

.bg-blobs { position: absolute; inset: 0; z-index: 0; overflow: hidden; pointer-events: none; border-radius: inherit; }
.blob { position: absolute; border-radius: 50%; filter: blur(70px); opacity: 0.3; }
.blob-1 { width: 560px; height: 560px; background: var(--cc-cyan); top: -200px; left: -140px; animation: cc-drift1 22s ease-in-out infinite; }
.blob-2 { width: 500px; height: 500px; background: var(--cc-violet); top: 5%; right: -200px; animation: cc-drift2 26s ease-in-out infinite; }
.blob-3 { width: 440px; height: 440px; background: var(--cc-rose); bottom: -200px; left: 25%; animation: cc-drift3 30s ease-in-out infinite; }
@keyframes cc-drift1 { 0%,100% { transform: translate(0,0) scale(1); } 50% { transform: translate(70px,50px) scale(1.15); } }
@keyframes cc-drift2 { 0%,100% { transform: translate(0,0) scale(1); } 50% { transform: translate(-60px,40px) scale(0.9); } }
@keyframes cc-drift3 { 0%,100% { transform: translate(0,0) scale(1); } 50% { transform: translate(50px,-35px) scale(1.1); } }
.grain { position: absolute; inset: 0; z-index: 1; pointer-events: none; opacity: 0.03; mix-blend-mode: overlay; border-radius: inherit;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='120' height='120'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E"); }
.spotlight { position: fixed; inset: 0; z-index: 2; pointer-events: none; mix-blend-mode: screen;
  background: radial-gradient(420px circle at var(--sx,50%) var(--sy,20%), rgba(45,212,240,0.07), transparent 70%); }

.shell { position: relative; z-index: 3; display: flex; flex-direction: column; gap: 14px; }

.ticker-bar { position: relative; z-index: 3; border-radius: 10px; background: rgba(var(--v-theme-on-surface),0.03); border: 1px solid var(--cc-border); overflow: hidden; margin-bottom: -2px; }
.ticker-track { display: flex; gap: 40px; white-space: nowrap; padding: 7px 14px; animation: cc-ticker 24s linear infinite; width: max-content; }
.ticker-track span { font-family: 'JetBrains Mono', monospace; font-size: 11px; color: rgba(var(--v-theme-on-surface),0.6); display: flex; align-items: center; gap: 7px; }
.ticker-track span::before { content: '●'; color: var(--cc-good); font-size: 7px; }
@keyframes cc-ticker { from { transform: translateX(0); } to { transform: translateX(-50%); } }

.header-bar { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; opacity: 0; transform: translateY(-14px); animation: cc-rise 0.55s cubic-bezier(.2,1.4,.4,1) 0.02s forwards; }
.header-title { font-size: 26px; font-weight: 700; letter-spacing: -0.01em;
  background: linear-gradient(100deg, var(--cc-cyan), var(--cc-violet) 45%, var(--cc-rose) 80%, var(--cc-cyan));
  background-size: 300% 100%; -webkit-background-clip: text; background-clip: text; color: transparent;
  animation: cc-hue 8s ease-in-out infinite; }
@keyframes cc-hue { 0%,100% { background-position: 0% 50%; } 50% { background-position: 100% 50%; } }
.header-sub { font-size: 13px; color: rgba(var(--v-theme-on-surface),0.55); margin-top: 4px; }
.refresh-row { display: flex; align-items: center; gap: 7px; font-size: 12px; color: rgba(var(--v-theme-on-surface),0.45); }
.refresh-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--cc-good); box-shadow: 0 0 0 3px rgba(52,229,168,0.18); animation: cc-pulse 2.4s ease-in-out infinite; }
.refresh-btn { background: none; border: none; color: inherit; cursor: pointer; display: flex; align-items: center; padding: 2px; border-radius: 6px; transition: background 0.2s ease; }
.refresh-btn:hover { background: rgba(var(--v-theme-on-surface),0.08); }
.refresh-btn:disabled { opacity: 0.4; cursor: default; }

.error-banner { background: var(--cc-crit-soft); border: 1px solid rgba(251,113,133,0.3); color: var(--cc-crit); padding: 10px 14px; border-radius: 10px; font-size: 13px; }
.empty-note { padding: 20px 4px; color: rgba(var(--v-theme-on-surface),0.4); font-size: 13px; }

.glow-card { position: relative; border-radius: 14px; padding: 1px;
  background: linear-gradient(115deg, rgba(45,212,240,0.5), rgba(168,85,247,0.45) 40%, rgba(251,90,160,0.45) 70%, rgba(45,212,240,0.5));
  background-size: 240% 240%; animation: cc-border-flow 9s ease infinite; }
.glow-card > .glow-inner { background: rgb(var(--v-theme-surface)); border-radius: 13px; height: 100%; }
@keyframes cc-border-flow { 0%,100% { background-position: 0% 30%; } 50% { background-position: 100% 70%; } }
.tilt-card { transition: transform 0.25s cubic-bezier(.16,1,.3,1); will-change: transform; }

.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
.stat-card { opacity: 0; transform: translateY(24px) scale(0.94); animation: cc-rise 0.6s cubic-bezier(.2,1.4,.4,1) forwards; }
.stat-card:nth-child(1) { animation-delay: 0.12s; } .stat-card:nth-child(2) { animation-delay: 0.22s; }
.stat-card:nth-child(3) { animation-delay: 0.32s; } .stat-card:nth-child(4) { animation-delay: 0.42s; }
.stat-body { padding: 18px 20px; display: flex; flex-direction: column; gap: 8px; }
.stat-icon { width: 30px; height: 30px; border-radius: 9px; display: grid; place-items: center; }
.stat-value { font-family: 'JetBrains Mono', monospace; font-size: 28px; font-weight: 600; letter-spacing: -0.01em; line-height: 1; transition: text-shadow 0.4s ease; }
.stat-value.crit { color: var(--cc-crit); }
.stat-value.done { text-shadow: 0 0 18px currentColor; }
.stat-label { font-size: 11.5px; letter-spacing: 0.06em; text-transform: uppercase; color: rgba(var(--v-theme-on-surface),0.4); }
.stat-breakdown { display: flex; gap: 10px; font-size: 12px; color: rgba(var(--v-theme-on-surface),0.55); margin-top: 2px; flex-wrap: wrap; }
.stat-breakdown b { color: inherit; font-family: 'JetBrains Mono', monospace; font-weight: 600; }
.stat-trend { font-size: 12px; font-weight: 600; display: flex; align-items: center; gap: 3px; color: var(--cc-good); }
.stat-trend.down { color: var(--cc-crit); }
.stat-mini-bars { display: flex; gap: 4px; align-items: flex-end; height: 22px; margin-top: 2px; }
.stat-mini-bars span { width: 8px; border-radius: 2px 2px 0 0; opacity: 0; transform: scaleY(0); transform-origin: bottom; }
.stat-warn-note { font-size: 12px; color: var(--cc-warn); text-decoration: none; }

.mid-grid { display: grid; grid-template-columns: 1.85fr 1fr; gap: 14px; align-items: stretch; }
.panel-body { padding: 20px 22px; height: 100%; }
.panel-head { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 4px; }
.panel-title { font-size: 15px; font-weight: 600; }
.panel-sub { font-size: 12px; color: rgba(var(--v-theme-on-surface),0.4); }
.chart-panel, .sidebar-panel, .funnel-panel, .activity-panel { opacity: 0; animation: cc-rise 0.6s cubic-bezier(.2,1.4,.4,1) forwards; }
.chart-panel { animation-delay: 0.5s; } .sidebar-panel { animation-delay: 0.58s; }
.funnel-panel { animation-delay: 0.66s; } .activity-panel { animation-delay: 0.82s; }

.legend { display: flex; gap: 14px; margin-top: 8px; margin-bottom: 6px; }
.legend-item { display: flex; align-items: center; gap: 6px; font-size: 12px; color: rgba(var(--v-theme-on-surface),0.55); cursor: pointer; user-select: none; transition: opacity 0.2s ease; }
.legend-item.off { opacity: 0.35; }
.legend-dot { width: 8px; height: 8px; border-radius: 2px; }
.chart-wrap { position: relative; }
svg { width: 100%; height: 220px; display: block; overflow: visible; }
.chart-tooltip { position: absolute; pointer-events: none; opacity: 0; background: rgb(var(--v-theme-surface)); border: 1px solid var(--cc-border);
  border-radius: 8px; padding: 8px 11px; font-size: 11.5px; white-space: nowrap; box-shadow: 0 12px 28px -10px rgba(0,0,0,0.6);
  transition: opacity 0.12s ease; transform: translate(-50%, -100%); }
.chart-tooltip :deep(.tt-date) { color: rgba(var(--v-theme-on-surface),0.4); font-family: 'JetBrains Mono', monospace; margin-bottom: 4px; }
.chart-tooltip :deep(.tt-row) { display: flex; align-items: center; gap: 6px; }
.chart-tooltip :deep(.tt-row + .tt-row) { margin-top: 2px; }
.chart-tooltip :deep(.tt-dot) { width: 7px; height: 7px; border-radius: 2px; flex-shrink: 0; }
.chart-tooltip :deep(.tt-val) { font-family: 'JetBrains Mono', monospace; font-weight: 600; margin-left: auto; }

.sidebar-body { display: flex; flex-direction: column; gap: 3px; }
.side-row { display: flex; align-items: center; justify-content: space-between; padding: 11px 0; border-top: 1px solid var(--cc-border); }
.side-row:first-of-type { border-top: none; }
.side-label { font-size: 12.5px; color: rgba(var(--v-theme-on-surface),0.55); }
.side-value { font-family: 'JetBrains Mono', monospace; font-size: 13.5px; font-weight: 600; }
.side-value.good { color: var(--cc-good); }

.table-scroll { overflow-x: auto; margin: 8px -4px 0; }
table.funnel { width: 100%; border-collapse: collapse; min-width: 720px; }
table.funnel th { text-align: left; font-size: 10.5px; letter-spacing: 0.06em; text-transform: uppercase; color: rgba(var(--v-theme-on-surface),0.4); font-weight: 600; padding: 8px 12px 10px; border-bottom: 1px solid var(--cc-border); }
table.funnel td { padding: 12px; border-top: 1px solid var(--cc-border); vertical-align: middle; }
tr.demo-row { cursor: pointer; transition: background 0.2s ease; opacity: 0; transform: translateX(-24px); animation: cc-slide 0.5s cubic-bezier(.2,1.3,.4,1) forwards; }
tr.demo-row:hover td { background: rgba(var(--v-theme-on-surface),0.05); }
tr.demo-row.at-limit td:first-child { box-shadow: inset 3px 0 0 var(--cc-crit); }
tr.demo-row.near-limit td:first-child { box-shadow: inset 3px 0 0 var(--cc-warn); }
tr.demo-row.at-limit { background: linear-gradient(90deg, var(--cc-crit-soft), transparent 40%); }
tr.demo-row.near-limit { background: linear-gradient(90deg, var(--cc-warn-soft), transparent 40%); }
.u-name { font-weight: 500; }
.u-name-sub { font-size: 11.5px; color: rgba(var(--v-theme-on-surface),0.4); margin-top: 1px; }
.chip { display: inline-block; font-size: 11px; font-weight: 600; padding: 3px 9px; border-radius: 100px; white-space: nowrap; }
.chip-role { background: rgba(var(--v-theme-on-surface),0.08); color: rgba(var(--v-theme-on-surface),0.6); border: 1px solid var(--cc-border); }
.chip-pending { background: var(--cc-warn-soft); color: var(--cc-warn); border: 1px solid rgba(251,191,36,0.3); }
.chip-approved { background: rgba(52,229,168,0.12); color: var(--cc-good); border: 1px solid rgba(52,229,168,0.3); }
.chip-rejected { background: var(--cc-crit-soft); color: var(--cc-crit); border: 1px solid rgba(251,113,133,0.3); }
.usage-cell { display: flex; gap: 8px; }
.u-bar { width: 40px; }
.u-bar-label { font-size: 9.5px; color: rgba(var(--v-theme-on-surface),0.4); text-align: center; margin-bottom: 3px; letter-spacing: 0.03em; }
.u-bar-track { height: 5px; border-radius: 4px; background: rgba(var(--v-theme-on-surface),0.1); overflow: hidden; }
.u-bar-fill { height: 100%; border-radius: 4px; width: 0%; transition: width 0.7s cubic-bezier(.16,1,.3,1); box-shadow: 0 0 7px currentColor; }
.u-bar-num { font-family: 'JetBrains Mono', monospace; font-size: 9.5px; color: rgba(var(--v-theme-on-surface),0.4); text-align: center; margin-top: 2px; }
.ai-dots { display: flex; gap: 5px; }
.ai-dot-group-label { font-size: 9.5px; color: rgba(var(--v-theme-on-surface),0.4); letter-spacing: 0.03em; margin-bottom: 4px; }
.ai-dot { width: 7px; height: 7px; border-radius: 50%; background: rgba(var(--v-theme-on-surface),0.12); display: inline-block; }
.ai-dot.on { background: var(--cc-violet); box-shadow: 0 0 6px var(--cc-violet); }

.activity-item { display: grid; grid-template-columns: 36px 1fr auto; gap: 14px; align-items: center; padding: 12px 6px; border-radius: 10px; border-top: 1px solid var(--cc-border);
  cursor: pointer; transition: background 0.2s ease, padding-left 0.2s ease; opacity: 0; transform: translateX(-20px); animation: cc-slide 0.5s cubic-bezier(.2,1.3,.4,1) forwards; }
.activity-item:first-child { border-top: none; }
.activity-item:hover { background: rgba(var(--v-theme-on-surface),0.05); padding-left: 12px; }
.user-avatar { width: 36px; height: 36px; border-radius: 10px; display: grid; place-items: center; font-family: 'Space Grotesk', sans-serif; font-weight: 700; font-size: 12.5px; color: #06141a;
  background: linear-gradient(145deg, var(--cc-cyan), var(--cc-violet) 130%); flex-shrink: 0; }
.activity-main { min-width: 0; }
.activity-user-row { display: flex; align-items: center; gap: 8px; }
.activity-user { font-size: 13px; font-weight: 600; }
.demo-tag { color: rgba(var(--v-theme-on-surface),0.4); font-weight: 400; }
.type-chip { font-size: 10px; font-weight: 600; padding: 2px 7px; border-radius: 100px; }
.type-ambient { background: rgba(45,212,240,0.13); color: var(--cc-cyan); }
.type-file { background: rgba(91,157,245,0.14); color: var(--cc-blue); }
.type-dictation { background: rgba(168,85,247,0.14); color: var(--cc-violet); }
.activity-title { font-size: 13.5px; margin-top: 3px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.activity-snippet { font-size: 12px; color: rgba(var(--v-theme-on-surface),0.4); margin-top: 1px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.activity-right { text-align: right; display: flex; flex-direction: column; align-items: flex-end; gap: 5px; }
.activity-time { font-size: 11px; color: rgba(var(--v-theme-on-surface),0.4); font-family: 'JetBrains Mono', monospace; }
.doc-chip { font-size: 10px; font-weight: 600; color: var(--cc-good); background: rgba(52,229,168,0.12); border: 1px solid rgba(52,229,168,0.28); padding: 2px 7px; border-radius: 100px; white-space: nowrap; }
.view-all { display: block; text-align: center; padding: 13px 0 4px; margin-top: 4px; font-size: 12.5px; font-weight: 600; color: rgba(var(--v-theme-on-surface),0.55); text-decoration: none; border-top: 1px solid var(--cc-border); transition: color 0.2s ease; }
.view-all:hover { color: var(--cc-cyan); }

@keyframes cc-rise { to { opacity: 1; transform: translateY(0) scale(1); } }
@keyframes cc-slide { to { opacity: 1; transform: translateX(0); } }
@keyframes cc-pulse { 0%, 100% { box-shadow: 0 0 0 3px rgba(52,229,168,0.18); } 50% { box-shadow: 0 0 0 6px rgba(52,229,168,0.05); } }

@media (prefers-reduced-motion: reduce) {
  .cc-root *, .cc-root *::before, .cc-root *::after { animation: none !important; }
  .header-bar, .stat-card, .glow-card, tr.demo-row, .activity-item { opacity: 1 !important; transform: none !important; }
}
@media (max-width: 1080px) { .stats-row { grid-template-columns: 1fr 1fr; } .mid-grid { grid-template-columns: 1fr; } }
@media (max-width: 620px) { .stats-row { grid-template-columns: 1fr; } .header-bar { flex-direction: column; align-items: flex-start; gap: 8px; } }
</style>
