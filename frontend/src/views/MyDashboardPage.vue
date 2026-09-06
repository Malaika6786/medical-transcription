<template>
  <div class="md-root">
    <div class="bg-blobs"><div class="blob blob-1"></div><div class="blob blob-2"></div></div>
    <div class="grain"></div>
    <div class="spotlight" ref="spotlightEl"></div>

    <div class="shell">
      <div v-if="viewingAsAdmin" class="admin-banner">
        <v-icon icon="mdi-shield-account" size="16" />
        <span>Viewing <b>{{ profileUser?.name || '…' }}</b>'s dashboard as superuser</span>
        <router-link to="/command-center" class="admin-banner-back">← Back to Command Center</router-link>
      </div>

      <div class="glow-card profile-header tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
        <div class="glow-inner profile-body">
          <div class="avatar">{{ initials }}<span class="status-ring"></span></div>
          <div class="profile-id">
            <div class="greeting">{{ viewingAsAdmin ? '' : greeting }}</div>
            <div class="profile-name-row"><span class="profile-name">{{ profileUser?.name }}</span><span class="role-chip" :class="{ 'role-chip-trial': isTrialView }">{{ formattedRole }}</span></div>
            <div class="profile-email">{{ profileUser?.email }}</div>
          </div>
          <div class="profile-meta">
            <div class="meta-block"><span class="meta-label">Member since</span><span class="meta-value">{{ memberSince }}</span></div>
            <div class="meta-block"><span class="meta-label">Last login</span><span class="meta-value">{{ lastLoginLabel }}</span></div>
          </div>
        </div>
      </div>

      <div v-if="isTrialView" class="trial-grid">
        <div class="glow-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body">
            <div class="panel-head"><h2 class="panel-title">Trial usage</h2><span class="panel-sub">{{ DEMO_TRIAL_LIMIT }} tries per feature</span></div>
            <div class="trial-feature-list">
              <div v-for="f in trialFeatures" :key="f.key" class="trial-feature-row">
                <span class="trial-feature-icon" :style="{ background: f.color + '24', color: f.color }"><v-icon :icon="f.icon" size="15" /></span>
                <div class="trial-feature-main">
                  <div class="trial-feature-top">
                    <span class="trial-feature-label">{{ f.label }}</span>
                    <span class="trial-feature-count" :class="{ maxed: isMaxedOut(f.key) }">{{ usageFor(f.key) }} / {{ DEMO_TRIAL_LIMIT }}</span>
                  </div>
                  <div class="trial-bar-track">
                    <div class="trial-bar-fill" :class="{ maxed: isMaxedOut(f.key) }"
                         :style="{ width: Math.min(100, (usageFor(f.key) / DEMO_TRIAL_LIMIT) * 100) + '%', background: f.color }"></div>
                  </div>
                </div>
                <span v-if="isMaxedOut(f.key)" class="trial-lock"><v-icon icon="mdi-lock" size="13" /> Locked</span>
                <span v-else class="trial-remaining">{{ remainingFor(f.key) }} left</span>
              </div>
            </div>
          </div>
        </div>

        <div class="glow-card tilt-card upgrade-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body upgrade-body">
            <div class="upgrade-badge"><v-icon icon="mdi-arrow-up-bold-circle-outline" size="14" /> Upgrade</div>
            <h2 class="panel-title upgrade-title">Unlock unlimited access</h2>
            <p class="upgrade-copy">A Doctor account removes the {{ DEMO_TRIAL_LIMIT }}-use trial cap — nothing else about your account changes.</p>
            <ul v-if="unlockableFeatures.length" class="upgrade-checklist">
              <li v-for="f in unlockableFeatures" :key="f.key"><v-icon icon="mdi-check-circle" size="14" /> Unlimited {{ f.label }}</li>
            </ul>
            <div v-if="trialStatusLabel === 'pending'" class="upgrade-status">
              <v-icon icon="mdi-clock-outline" size="15" /> Your request for {{ requestedRoleLabel }} access is under review.
            </div>
            <a v-else :href="contactHref" class="upgrade-cta">
              <v-icon icon="mdi-email-fast-outline" size="16" /> Request Doctor access
            </a>
          </div>
        </div>
      </div>

      <div class="stats-row">
        <div class="glow-card stat-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(45,212,240,0.14);color:var(--md-cyan)"><v-icon icon="mdi-content-save-all" size="15" /></div>
            <span class="stat-value" :data-count="totalSessions">0</span>
            <span class="stat-label">Total sessions saved</span>
          </div>
        </div>

        <div class="glow-card stat-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(91,157,245,0.14);color:var(--md-blue)"><v-icon icon="mdi-calendar-week" size="15" /></div>
            <span class="stat-value" :data-count="sessionsThisWeek">0</span>
            <span class="stat-label">Sessions this week</span>
            <span v-if="weeklyTrendPct !== null" class="stat-trend" :class="{ down: weeklyTrendPct < 0 }">
              <v-icon :icon="weeklyTrendPct < 0 ? 'mdi-arrow-down' : 'mdi-arrow-up'" size="11" />{{ Math.abs(weeklyTrendPct) }}%
            </span>
          </div>
        </div>

        <div class="glow-card stat-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(168,85,247,0.14);color:var(--md-violet)"><v-icon icon="mdi-file-document-outline" size="15" /></div>
            <span class="stat-value" :data-count="documentsGenerated">0</span>
            <span class="stat-label">Documents generated</span>
          </div>
        </div>

        <div class="glow-card stat-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner stat-body">
            <div class="stat-icon" style="background:rgba(251,191,36,0.14);color:var(--md-warn)"><v-icon icon="mdi-star-outline" size="15" /></div>
            <span v-if="mostUsedFeature" class="stat-pill">{{ typeLabel(mostUsedFeature) }}</span>
            <span v-else class="stat-pill" style="opacity:0.4">—</span>
            <span class="stat-label">Most-used feature</span>
          </div>
        </div>
      </div>

      <div v-if="quickActions.length" class="sec-label"><h2>Quick actions</h2></div>
      <div v-if="quickActions.length" class="quick-actions">
        <div v-for="qa in quickActions" :key="qa.path" class="glow-card qa-card tilt-card" :class="{ 'qa-locked': isTrialView && isMaxedOut(qa.feature) }" @mousemove="onTilt" @mouseleave="onTiltLeave" @click="onQuickAction(qa)">
          <div class="glow-inner qa-body">
            <span class="qa-shine" :style="{ background: `radial-gradient(220px 120px at 0% 0%, ${qa.glow}, transparent 70%)` }"></span>
            <span class="qa-icon" :style="{ background: qa.iconBg, color: qa.color }"><v-icon :icon="isTrialView && isMaxedOut(qa.feature) ? 'mdi-lock' : qa.icon" size="20" /></span>
            <span><div class="qa-label">{{ qa.label }}</div><div class="qa-sub">{{ qa.sub }}</div></span>
            <span v-if="isTrialView" class="qa-trial-badge" :class="{ maxed: isMaxedOut(qa.feature) }">{{ isMaxedOut(qa.feature) ? 'Locked' : `${remainingFor(qa.feature)} left` }}</span>
          </div>
        </div>
      </div>

      <div class="mid-grid">
        <div class="glow-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body">
            <div class="panel-head"><h2 class="panel-title">Recent sessions</h2><span class="panel-sub">Last {{ recentSessions.length }} of {{ totalSessions }}</span></div>
            <div v-if="recentSessions.length === 0" class="empty-note">{{ viewingAsAdmin ? 'No sessions saved yet.' : 'No sessions saved yet — start one from Quick actions above.' }}</div>
            <div v-else>
              <div v-for="(s, i) in recentSessions" :key="s.id" class="activity-item" :style="{ animationDelay: (0.06 * i) + 's' }" @click="!viewingAsAdmin && router.push('/saved-sessions')">
                <span class="type-badge" :class="typeClass(s.type)"><v-icon :icon="typeIcon(s.type)" size="16" /></span>
                <div class="activity-main">
                  <div class="activity-title">{{ s.title }}</div>
                  <div class="activity-snippet">"...{{ snippet(s.transcript) }}..."</div>
                </div>
                <div class="activity-right"><span class="activity-time">{{ relativeTime(s.createdAt) }}</span><span v-if="s.document" class="doc-chip">✓ Document</span></div>
              </div>
            </div>
            <router-link v-if="!viewingAsAdmin" to="/saved-sessions" class="view-all">View all sessions →</router-link>
          </div>
        </div>

        <div class="glow-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body">
            <div class="panel-head"><h2 class="panel-title">Activity breakdown</h2></div>
            <div v-if="totalSessions === 0" class="empty-note">Nothing to break down yet.</div>
            <div v-else class="donut-wrap">
              <div class="donut-figure">
                <svg viewBox="0 0 100 100">
                  <circle cx="50" cy="50" r="40" fill="none" style="stroke:rgba(var(--v-theme-on-surface),0.12)" stroke-width="15"/>
                  <circle v-for="seg in donutSegments" :key="seg.type" class="donut-seg" :data-pct="seg.pct"
                          cx="50" cy="50" r="40" :stroke="seg.color" stroke-dasharray="251.2" stroke-dashoffset="251.2"
                          :style="{ filter: `drop-shadow(0 0 5px ${seg.color})` }"/>
                </svg>
                <div class="donut-center"><span class="n">{{ totalSessions }}</span><span class="l">sessions</span></div>
              </div>
              <div class="legend">
                <div v-for="seg in donutSegments" :key="seg.type" class="legend-row">
                  <span class="legend-dot" :style="{ background: seg.color }"></span>
                  <span class="legend-name">{{ typeLabel(seg.type) }}</span>
                  <span class="legend-val">{{ seg.count }} · {{ seg.pct }}%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="isTrialView" class="mid-grid">
        <div class="glow-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body">
            <div class="panel-head"><h2 class="panel-title">Clinical documents</h2><span class="panel-sub">{{ clinicalDocuments.length }} generated</span></div>
            <div v-if="clinicalDocuments.length === 0" class="empty-note">No documents generated yet — try Document generation from a saved session.</div>
            <div v-else>
              <div v-for="(s, i) in clinicalDocuments" :key="s.id" class="activity-item" :style="{ animationDelay: (0.06 * i) + 's' }" @click="router.push('/saved-sessions')">
                <span class="type-badge" :class="typeClass(s.type)"><v-icon icon="mdi-file-document-check-outline" size="16" /></span>
                <div class="activity-main">
                  <div class="activity-title">{{ s.title }}</div>
                  <div class="activity-snippet">{{ typeLabel(s.type) }} session</div>
                </div>
                <div class="activity-right"><span class="activity-time">{{ relativeTime(s.createdAt) }}</span><span class="doc-chip">View →</span></div>
              </div>
            </div>
            <router-link to="/saved-sessions" class="view-all">View all documents →</router-link>
          </div>
        </div>

        <div class="glow-card tilt-card" @mousemove="onTilt" @mouseleave="onTiltLeave">
          <div class="glow-inner panel-body">
            <div class="panel-head"><h2 class="panel-title">Help &amp; support</h2></div>
            <div class="help-links">
              <router-link to="/demo-guide" class="help-link"><v-icon icon="mdi-compass-outline" size="17" /> Demo guide</router-link>
              <router-link to="/docs" class="help-link"><v-icon icon="mdi-book-open-variant-outline" size="17" /> Documentation</router-link>
              <a :href="contactHref" class="help-link"><v-icon icon="mdi-email-outline" size="17" /> Contact an administrator</a>
            </div>
          </div>
        </div>
      </div>
    </div>

    <v-dialog v-model="limitDialogOpen" max-width="420">
      <div class="glow-card limit-dialog">
        <div class="glow-inner limit-dialog-body">
          <div class="limit-icon"><v-icon icon="mdi-lock-outline" size="26" /></div>
          <h2 class="limit-title">You've used all {{ DEMO_TRIAL_LIMIT }} trial attempts</h2>
          <p class="limit-copy">You've reached the trial limit for <b>{{ limitDialogFeatureLabel }}</b>. Upgrade to a Doctor account to keep using it — everything else about your account stays the same.</p>
          <div class="limit-actions">
            <a :href="contactHref" class="upgrade-cta">Request Doctor access</a>
            <v-btn variant="text" size="small" @click="limitDialogOpen = false">Back to dashboard</v-btn>
          </div>
        </div>
      </div>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { currentUser, hasPermission, isDemoAccount } from '@/stores/auth'
import { userSessions, loadSessions, type SavedSession } from '@/stores/sessions'
import { DEMO_FEATURES, DEMO_TRIAL_LIMIT, loadDemoUsage, usageFor, remainingFor, isMaxedOut } from '@/stores/demoUsage'
import api from '@/services/api'

// SUPPORT_EMAIL is the seeded superuser account (pgstore.EnsureSeed) — the
// one real contact point in a small internal deployment like this, not a
// fabricated support address.
const SUPPORT_EMAIL = 'admin@xstek.net'

const router = useRouter()
const route = useRoute()
const spotlightEl = ref<HTMLElement | null>(null)
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

// --- superuser drill-down: /user-dashboard/:userId reuses this exact
// component with a different data source, per the Command Center's
// "same dashboard they'd see, not a separate admin view" design.
const targetUserId = computed(() => route.params.userId as string | undefined)
const viewingAsAdmin = computed(() => !!targetUserId.value)
const otherUser = ref<{ name: string; email: string; roles: string[]; createdAt: string; lastLogin?: string } | null>(null)
const otherSessions = ref<SavedSession[]>([])

const profileUser = computed(() => (viewingAsAdmin.value ? otherUser.value : currentUser.value))
const sessions = computed(() => (viewingAsAdmin.value ? otherSessions.value : userSessions.value))

const initials = computed(() => {
  const name = profileUser.value?.name || ''
  const parts = name.replace(/^Dr\.\s*/i, '').trim().split(/\s+/)
  return parts.slice(0, 2).map(p => p[0]?.toUpperCase() || '').join('') || '?'
})
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 12) return 'Good morning'
  if (h < 18) return 'Good afternoon'
  return 'Good evening'
})
const formattedRole = computed(() => {
  const role = profileUser.value?.roles?.[0]
  if (role === 'superuser') return 'Super User'
  if (role === 'doctor') return 'Doctor'
  if (role === 'admin') return 'Admin'
  if (role === 'user') return 'Trial Account'
  return role ? role.charAt(0).toUpperCase() + role.slice(1) : 'User'
})

// Trial dashboard: only for a demo ("user" role) account looking at its own
// dashboard — never in the superuser drill-down view.
const isTrialView = computed(() => !viewingAsAdmin.value && isDemoAccount.value)
const trialFeatures = DEMO_FEATURES
const trialStatusLabel = computed(() => {
  const status = currentUser.value?.status
  if (status === 'pending') return 'pending'
  return 'active'
})
const requestedRoleLabel = computed(() => {
  const r = currentUser.value?.requestedRole
  if (!r) return 'Doctor'
  return r.charAt(0).toUpperCase() + r.slice(1)
})
const limitDialogOpen = ref(false)
const limitDialogFeature = ref<string | null>(null)
function openLimitDialog(featureKey: string) {
  limitDialogFeature.value = featureKey
  limitDialogOpen.value = true
}
const limitDialogFeatureLabel = computed(() => {
  const f = DEMO_FEATURES.find(f => f.key === limitDialogFeature.value)
  return f?.label || 'this feature'
})
const contactHref = computed(() => `mailto:${SUPPORT_EMAIL}?subject=${encodeURIComponent('Requesting full Doctor access')}`)

const clinicalDocuments = computed(() => sessions.value.filter(s => !!s.document).slice(0, 5))

// Only claim an upgrade unlocks features this account is actually granted
// today — role permissions are admin-configurable (Role Management), so
// this can't be hardcoded.
const FEATURE_PERM: Record<string, string> = {
  ambient: 'ambient.access',
  file_transcription: 'file_transcription.access',
  dictation: 'dictation.access',
}
const unlockableFeatures = computed(() => trialFeatures.filter(f => {
  const perm = FEATURE_PERM[f.key]
  return perm ? hasPermission(perm) : false
}))
const memberSince = computed(() => {
  const c = profileUser.value?.createdAt
  return c ? new Date(c).toLocaleDateString(undefined, { month: 'short', year: 'numeric' }) : '—'
})
const lastLoginLabel = computed(() => {
  const l = profileUser.value?.lastLogin
  return l ? relativeTime(l) : 'This session'
})

const totalSessions = computed(() => sessions.value.length)

const sessionsThisWeek = computed(() => sessions.value.filter(s => daysAgo(s.createdAt) < 7).length)
const sessionsPrevWeek = computed(() => sessions.value.filter(s => { const d = daysAgo(s.createdAt); return d >= 7 && d < 14 }).length)
const weeklyTrendPct = computed<number | null>(() => {
  if (sessionsPrevWeek.value === 0) return null
  return Math.round(((sessionsThisWeek.value - sessionsPrevWeek.value) / sessionsPrevWeek.value) * 100)
})
const documentsGenerated = computed(() => sessions.value.filter(s => !!s.document).length)

const typeCounts = computed(() => {
  const m: Record<string, number> = {}
  sessions.value.forEach(s => { m[s.type] = (m[s.type] || 0) + 1 })
  return m
})
const mostUsedFeature = computed(() => {
  const entries = Object.entries(typeCounts.value)
  if (entries.length === 0) return null
  return entries.sort((a, b) => b[1] - a[1])[0][0]
})

const TYPE_COLOR: Record<string, string> = { ambient: '#2dd4f0', 'file-transcription': '#5b9df5', dictation: '#a855f7', 'embedded-assistant': '#fb5aa0' }
const donutSegments = computed(() => {
  const total = totalSessions.value
  if (total === 0) return []
  return Object.entries(typeCounts.value)
    .sort((a, b) => b[1] - a[1])
    .map(([type, count]) => ({ type, count, pct: Math.round((count / total) * 100), color: TYPE_COLOR[type] || '#8c9bb0' }))
})

const recentSessions = computed(() => sessions.value.slice(0, 6))

const quickActions = computed(() => {
  // Can't start a session as someone else — no quick actions in drill-down view.
  if (viewingAsAdmin.value) return []
  const all = [
    { path: '/ambient-session', label: 'Start ambient session', sub: 'Listens in the background during the visit', icon: 'mdi-broadcast', color: '#2dd4f0', iconBg: 'rgba(45,212,240,0.14)', glow: 'rgba(45,212,240,0.16)', perm: 'ambient.access', feature: 'ambient' },
    { path: '/async-transcription', label: 'Upload a file', sub: 'Transcribe an existing recording', icon: 'mdi-file-upload', color: '#5b9df5', iconBg: 'rgba(91,157,245,0.14)', glow: 'rgba(91,157,245,0.16)', perm: 'file_transcription.access', feature: 'file_transcription' },
    { path: '/dictation', label: 'Dictate a note', sub: 'Speak a note directly into a document', icon: 'mdi-microphone-message', color: '#a855f7', iconBg: 'rgba(168,85,247,0.14)', glow: 'rgba(168,85,247,0.16)', perm: 'dictation.access', feature: 'dictation' }
  ]
  return all.filter(a => hasPermission(a.perm))
})
function onQuickAction(qa: { path: string; feature: string }) {
  if (isTrialView.value && isMaxedOut(qa.feature)) { openLimitDialog(qa.feature); return }
  router.push(qa.path)
}

function daysAgo(iso: string) { return (Date.now() - new Date(iso).getTime()) / 86400000 }
function relativeTime(iso: string) {
  const secs = Math.round((Date.now() - new Date(iso).getTime()) / 1000)
  if (secs < 60) return 'just now'
  const mins = Math.round(secs / 60)
  if (mins < 60) return `${mins}m ago`
  const hours = Math.round(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  if (days === 1) return 'yesterday'
  if (days < 30) return `${days} days ago`
  return new Date(iso).toLocaleDateString()
}
function snippet(transcript: string) {
  const t = (transcript || '').trim().replace(/\s+/g, ' ')
  return t.length > 90 ? t.slice(0, 90) + '…' : t
}
function typeClass(type: string) {
  if (type === 'ambient') return 'type-ambient'
  if (type === 'file-transcription') return 'type-file'
  if (type === 'dictation') return 'type-dictation'
  return 'type-assistant'
}
function typeIcon(type: string) {
  if (type === 'ambient') return 'mdi-broadcast'
  if (type === 'file-transcription') return 'mdi-file-upload'
  if (type === 'dictation') return 'mdi-microphone-message'
  return 'mdi-robot-outline'
}
function typeLabel(type: string) {
  if (type === 'ambient') return 'Ambient'
  if (type === 'file-transcription') return 'File'
  if (type === 'dictation') return 'Dictation'
  if (type === 'embedded-assistant') return 'Assistant'
  return type
}

function onTilt(e: MouseEvent) {
  if (reducedMotion) return
  const card = e.currentTarget as HTMLElement
  const r = card.getBoundingClientRect()
  const px = (e.clientX - r.left) / r.width - 0.5
  const py = (e.clientY - r.top) / r.height - 0.5
  card.style.transform = `perspective(700px) rotateX(${-py * 5}deg) rotateY(${px * 6}deg) translateZ(2px)`
}
function onTiltLeave(e: MouseEvent) { (e.currentTarget as HTMLElement).style.transform = '' }
function onSpotlightMove(e: MouseEvent) {
  spotlightEl.value?.style.setProperty('--sx', e.clientX + 'px')
  spotlightEl.value?.style.setProperty('--sy', e.clientY + 'px')
}
if (typeof window !== 'undefined' && !reducedMotion) {
  window.addEventListener('mousemove', onSpotlightMove)
  onUnmounted(() => window.removeEventListener('mousemove', onSpotlightMove))
}

function animateCounts() {
  document.querySelectorAll<HTMLElement>('.md-root .stat-value[data-count]').forEach((el, i) => {
    const target = parseInt(el.dataset.count || '0', 10)
    setTimeout(() => countUp(el, target, 1100), 300 + i * 120)
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
    else { el.textContent = String(target); el.classList.add('done'); setTimeout(() => el.classList.remove('done'), 900) }
  }
  requestAnimationFrame(step)
}

function drawDonut() {
  const CIRC = 251.2
  let cursor = 0
  const segs = document.querySelectorAll<SVGCircleElement>('.md-root .donut-seg')
  segs.forEach((seg, i) => {
    const pct = parseFloat(seg.dataset.pct || '0') / 100
    const len = pct * CIRC
    seg.style.strokeDasharray = `${len} ${CIRC - len}`
    seg.style.strokeDashoffset = String(CIRC - cursor)
    cursor += len
    if (!reducedMotion) {
      seg.style.transition = 'stroke-dashoffset 1s cubic-bezier(.2,1.2,.3,1)'
      seg.style.strokeDasharray = `0 ${CIRC}`
      setTimeout(() => { seg.style.strokeDasharray = `${len} ${CIRC - len}` }, 750 + i * 160)
    }
  })
}

onMounted(async () => {
  if (viewingAsAdmin.value) {
    const id = targetUserId.value!
    try {
      const [usersRes, sessionsRes] = await Promise.all([
        api.get('/users'),
        api.get(`/admin/sessions/${id}`)
      ])
      const list = usersRes.data.users || []
      otherUser.value = list.find((u: any) => u.id === id) || null
      otherSessions.value = sessionsRes.data.sessions || []
    } catch (err) {
      console.error('Failed to load user dashboard:', err)
    }
  } else if (userSessions.value.length === 0) {
    await loadSessions()
  }
  if (isTrialView.value) {
    await loadDemoUsage()
  }
  await nextTick()
  animateCounts()
  drawDonut()
})
</script>

<style scoped>
.md-root {
  --md-cyan: #2dd4f0; --md-blue: #5b9df5; --md-violet: #a855f7; --md-rose: #fb5aa0;
  --md-good: #34e5a8; --md-warn: #fbbf24;
  --md-border: rgba(var(--v-theme-on-surface),0.08);
  position: relative;
  font-variant-numeric: tabular-nums;
  font-family: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  color: rgb(var(--v-theme-on-background));
  margin: -16px; padding: 16px;
  overflow: hidden;
  border-radius: 12px;
}
.md-root h1, .md-root h2 { font-family: 'Space Grotesk', sans-serif; margin: 0; }

.bg-blobs { position: absolute; inset: 0; z-index: 0; overflow: hidden; pointer-events: none; border-radius: inherit; }
.blob { position: absolute; border-radius: 50%; filter: blur(70px); opacity: 0.3; }
.blob-1 { width: 520px; height: 520px; background: var(--md-cyan); top: -200px; left: -140px; animation: md-drift1 22s ease-in-out infinite; }
.blob-2 { width: 460px; height: 460px; background: var(--md-violet); bottom: -200px; right: -180px; animation: md-drift2 26s ease-in-out infinite; }
@keyframes md-drift1 { 0%,100% { transform: translate(0,0); } 50% { transform: translate(60px,50px); } }
@keyframes md-drift2 { 0%,100% { transform: translate(0,0); } 50% { transform: translate(-50px,40px); } }
.grain { position: absolute; inset: 0; z-index: 1; pointer-events: none; opacity: 0.03; mix-blend-mode: overlay; border-radius: inherit;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='120' height='120'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E"); }
.spotlight { position: fixed; inset: 0; z-index: 2; pointer-events: none; mix-blend-mode: screen;
  background: radial-gradient(420px circle at var(--sx,50%) var(--sy,20%), rgba(45,212,240,0.07), transparent 70%); }

.shell { position: relative; z-index: 3; display: flex; flex-direction: column; gap: 14px; }

.glow-card { position: relative; border-radius: 15px; padding: 1px;
  background: linear-gradient(115deg, rgba(45,212,240,0.5), rgba(168,85,247,0.45) 40%, rgba(251,90,160,0.45) 70%, rgba(45,212,240,0.5));
  background-size: 240% 240%; animation: md-border-flow 9s ease infinite; }
.glow-card > .glow-inner { background: rgb(var(--v-theme-surface)); border-radius: 14px; height: 100%; }
@keyframes md-border-flow { 0%,100% { background-position: 0% 30%; } 50% { background-position: 100% 70%; } }
.tilt-card { transition: transform 0.25s cubic-bezier(.16,1,.3,1); will-change: transform; cursor: default; }

.admin-banner {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  padding: 10px 16px; border-radius: 12px;
  background: rgba(45,212,240,0.1); border: 1px solid rgba(45,212,240,0.3);
  color: var(--md-cyan); font-size: 13px; font-weight: 500;
  opacity: 0; animation: md-rise 0.5s cubic-bezier(.2,1.4,.4,1) forwards;
}
.admin-banner b { font-weight: 700; }
.admin-banner-back { margin-left: auto; color: inherit; font-size: 12.5px; font-weight: 600; text-decoration: none; opacity: 0.85; }
.admin-banner-back:hover { opacity: 1; text-decoration: underline; }

.profile-header { opacity: 0; transform: translateY(-14px); animation: md-rise 0.7s cubic-bezier(.2,1.4,.4,1) 0.05s forwards; }
.profile-body { padding: 20px 24px; display: flex; align-items: center; gap: 18px; }
.avatar { position: relative; width: 50px; height: 50px; border-radius: 14px; background: linear-gradient(145deg, var(--md-cyan), var(--md-violet) 130%); display: grid; place-items: center; font-family: 'Space Grotesk', sans-serif; font-weight: 700; font-size: 16px; color: #0a0713; flex-shrink: 0; }
.status-ring { position: absolute; bottom: -2px; right: -2px; width: 12px; height: 12px; border-radius: 50%; background: var(--md-good); border: 3px solid rgb(var(--v-theme-surface)); box-shadow: 0 0 0 2px rgba(52,229,168,0.2); }
.profile-id { flex: 1; min-width: 0; }
.greeting { font-size: 11.5px; color: rgba(var(--v-theme-on-surface),0.4); margin-bottom: 2px; }
.profile-name-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.profile-name { font-size: 19px; font-weight: 700; letter-spacing: -0.01em;
  background: linear-gradient(100deg, var(--md-cyan), var(--md-violet) 45%, var(--md-rose) 80%, var(--md-cyan));
  background-size: 300% 100%; -webkit-background-clip: text; background-clip: text; color: transparent; animation: md-hue 8s ease-in-out infinite; }
@keyframes md-hue { 0%,100% { background-position: 0% 50%; } 50% { background-position: 100% 50%; } }
.role-chip { font-size: 11px; font-weight: 700; letter-spacing: 0.04em; text-transform: uppercase; padding: 3px 10px; border-radius: 100px; background: rgba(45,212,240,0.14); color: var(--md-cyan); border: 1px solid rgba(45,212,240,0.3); }
.profile-email { color: rgba(var(--v-theme-on-surface),0.5); font-size: 13px; margin-top: 3px; }
.profile-meta { display: flex; gap: 18px; flex-shrink: 0; text-align: right; }
.meta-block { display: flex; flex-direction: column; gap: 2px; }
.meta-label { font-size: 10.5px; letter-spacing: 0.07em; text-transform: uppercase; color: rgba(var(--v-theme-on-surface),0.35); }
.meta-value { font-size: 13px; color: rgba(var(--v-theme-on-surface),0.55); }

.role-chip-trial { background: rgba(251,191,36,0.14); color: var(--md-warn); border-color: rgba(251,191,36,0.3); }

.trial-grid { display: grid; grid-template-columns: 1.3fr 1fr; gap: 14px; align-items: stretch;
  opacity: 0; animation: md-rise 0.7s cubic-bezier(.2,1.4,.4,1) 0.12s forwards; }
.trial-feature-list { display: flex; flex-direction: column; gap: 12px; margin-top: 6px; }
.trial-feature-row { display: grid; grid-template-columns: 30px 1fr auto; gap: 12px; align-items: center; }
.trial-feature-icon { width: 30px; height: 30px; border-radius: 9px; display: grid; place-items: center; flex-shrink: 0; }
.trial-feature-main { min-width: 0; }
.trial-feature-top { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 5px; }
.trial-feature-label { font-size: 13px; font-weight: 500; }
.trial-feature-count { font-family: 'JetBrains Mono', monospace; font-size: 11.5px; color: rgba(var(--v-theme-on-surface),0.4); }
.trial-feature-count.maxed { color: var(--md-rose); font-weight: 600; }
.trial-bar-track { height: 6px; border-radius: 100px; background: var(--md-border); overflow: hidden; }
.trial-bar-fill { height: 100%; border-radius: 100px; transition: width 0.6s cubic-bezier(.2,1,.3,1); }
.trial-bar-fill.maxed { background: var(--md-rose) !important; }
.trial-lock { font-size: 11px; font-weight: 600; color: var(--md-rose); display: flex; align-items: center; gap: 3px; white-space: nowrap; }
.trial-remaining { font-size: 11px; color: rgba(var(--v-theme-on-surface),0.4); white-space: nowrap; }

.upgrade-card > .glow-inner { background: linear-gradient(160deg, rgba(45,212,240,0.08), rgba(168,85,247,0.08)); }
.upgrade-body { display: flex; flex-direction: column; gap: 10px; }
.upgrade-badge { align-self: flex-start; font-size: 10.5px; font-weight: 700; letter-spacing: 0.05em; text-transform: uppercase; display: flex; align-items: center; gap: 4px; color: var(--md-violet); background: rgba(168,85,247,0.14); border: 1px solid rgba(168,85,247,0.3); padding: 3px 10px; border-radius: 100px; }
.upgrade-title { margin-top: 2px; }
.upgrade-copy { font-size: 13px; color: rgba(var(--v-theme-on-surface),0.55); margin: 0; line-height: 1.5; }
.upgrade-checklist { list-style: none; padding: 0; margin: 2px 0 4px; display: flex; flex-direction: column; gap: 7px; }
.upgrade-checklist li { display: flex; align-items: center; gap: 8px; font-size: 12.5px; color: rgba(var(--v-theme-on-surface),0.7); }
.upgrade-checklist li .v-icon { color: var(--md-good); }
.upgrade-status { font-size: 12.5px; color: var(--md-warn); display: flex; align-items: center; gap: 6px; background: rgba(251,191,36,0.1); border: 1px solid rgba(251,191,36,0.25); border-radius: 10px; padding: 9px 12px; }
.upgrade-cta { align-self: flex-start; display: flex; align-items: center; gap: 7px; font-size: 13px; font-weight: 700; color: #0a0713; background: linear-gradient(100deg, var(--md-cyan), var(--md-violet)); padding: 9px 18px; border-radius: 10px; text-decoration: none; transition: transform 0.2s ease, box-shadow 0.2s ease; }
.upgrade-cta:hover { transform: translateY(-1px); box-shadow: 0 6px 18px rgba(168,85,247,0.3); }

.qa-locked { opacity: 0.72; }
.qa-trial-badge { position: absolute; top: 10px; right: 12px; font-size: 10px; font-weight: 700; letter-spacing: 0.03em; color: rgba(var(--v-theme-on-surface),0.4); }
.qa-trial-badge.maxed { color: var(--md-rose); }

.help-links { display: flex; flex-direction: column; gap: 4px; margin-top: 4px; }
.help-link { display: flex; align-items: center; gap: 10px; padding: 11px 8px; border-radius: 10px; color: rgb(var(--v-theme-on-surface)); text-decoration: none; font-size: 13.5px; font-weight: 500; transition: background 0.2s ease, padding-left 0.2s ease; }
.help-link .v-icon { color: var(--md-cyan); }
.help-link:hover { background: rgba(var(--v-theme-on-surface),0.05); padding-left: 12px; }

.limit-dialog {
  /* v-dialog content is teleported to <body>, outside .md-root, so the
     custom properties defined there aren't in scope — redeclare them. */
  --md-cyan: #2dd4f0; --md-blue: #5b9df5; --md-violet: #a855f7; --md-rose: #fb5aa0;
  --md-good: #34e5a8; --md-warn: #fbbf24;
}
.limit-dialog-body { padding: 26px 26px 22px; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 12px; }
.limit-icon { width: 52px; height: 52px; border-radius: 50%; background: rgba(251,90,160,0.14); color: var(--md-rose); display: grid; place-items: center; }
.limit-title { font-size: 16px; }
.limit-copy { font-size: 13.5px; color: rgba(var(--v-theme-on-surface),0.55); line-height: 1.5; margin: 0; }
.limit-actions { display: flex; flex-direction: column; align-items: center; gap: 8px; margin-top: 6px; width: 100%; }

.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
.stat-card { opacity: 0; transform: translateY(24px) scale(0.94); animation: md-rise 0.6s cubic-bezier(.2,1.4,.4,1) forwards; }
.stat-card:nth-child(1) { animation-delay: 0.16s; } .stat-card:nth-child(2) { animation-delay: 0.26s; }
.stat-card:nth-child(3) { animation-delay: 0.36s; } .stat-card:nth-child(4) { animation-delay: 0.46s; }
.stat-body { padding: 18px 20px; display: flex; flex-direction: column; gap: 8px; }
.stat-icon { width: 30px; height: 30px; border-radius: 9px; display: grid; place-items: center; }
.stat-value { font-family: 'JetBrains Mono', monospace; font-size: 28px; font-weight: 600; letter-spacing: -0.01em; line-height: 1; transition: text-shadow 0.4s ease; }
.stat-value.done { text-shadow: 0 0 18px currentColor; }
.stat-label { font-size: 11.5px; letter-spacing: 0.06em; text-transform: uppercase; color: rgba(var(--v-theme-on-surface),0.4); }
.stat-trend { font-size: 12px; font-weight: 600; display: flex; align-items: center; gap: 3px; color: var(--md-good); }
.stat-trend.down { color: #fb7185; }
.stat-pill { align-self: flex-start; font-size: 13.5px; font-weight: 700; padding: 5px 12px; border-radius: 100px; background: rgba(45,212,240,0.14); color: var(--md-cyan); border: 1px solid rgba(45,212,240,0.3); }

.sec-label { margin: 4px 2px -2px; }
.sec-label h2 { font-size: 14px; font-weight: 600; }
.quick-actions { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.qa-card { opacity: 0; transform: translateY(24px) scale(0.95); animation: md-rise 0.65s cubic-bezier(.2,1.4,.4,1) forwards; cursor: pointer; }
.qa-card:nth-child(1) { animation-delay: 0.56s; } .qa-card:nth-child(2) { animation-delay: 0.64s; } .qa-card:nth-child(3) { animation-delay: 0.72s; }
.qa-body { padding: 18px 20px; display: flex; align-items: center; gap: 13px; position: relative; overflow: hidden; }
.qa-icon { width: 42px; height: 42px; border-radius: 12px; display: grid; place-items: center; flex-shrink: 0; }
.qa-label { font-weight: 700; font-size: 14.5px; }
.qa-sub { font-size: 12px; color: rgba(var(--v-theme-on-surface),0.4); margin-top: 1px; }
.qa-shine { position: absolute; inset: 0; opacity: 0; transition: opacity 0.3s ease; pointer-events: none; }
.qa-card:hover .qa-shine { opacity: 1; }

.mid-grid { display: grid; grid-template-columns: 1.6fr 1fr; gap: 14px; align-items: stretch; }
.mid-grid > .glow-card:first-child { opacity: 0; animation: md-rise 0.7s cubic-bezier(.2,1.4,.4,1) 0.8s forwards; }
.mid-grid > .glow-card:last-child { opacity: 0; animation: md-rise 0.7s cubic-bezier(.2,1.4,.4,1) 0.9s forwards; }
.panel-body { padding: 20px 22px; height: 100%; }
.panel-head { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 6px; }
.panel-title { font-size: 15px; font-weight: 600; }
.panel-sub { font-size: 12px; color: rgba(var(--v-theme-on-surface),0.4); }
.empty-note { padding: 16px 4px; color: rgba(var(--v-theme-on-surface),0.4); font-size: 13px; }

.activity-item { display: grid; grid-template-columns: 34px 1fr auto; gap: 14px; align-items: center; padding: 13px 8px; border-radius: 10px; border-top: 1px solid var(--md-border);
  cursor: pointer; transition: background 0.2s ease, padding-left 0.2s ease; opacity: 0; transform: translateX(-20px); animation: md-slide 0.55s cubic-bezier(.2,1.3,.4,1) forwards; }
.activity-item:first-child { border-top: none; }
.activity-item:hover { background: rgba(var(--v-theme-on-surface),0.05); padding-left: 12px; }
.type-badge { width: 34px; height: 34px; border-radius: 9px; display: grid; place-items: center; flex-shrink: 0; }
.type-ambient { background: rgba(45,212,240,0.13); color: var(--md-cyan); }
.type-file { background: rgba(91,157,245,0.14); color: var(--md-blue); }
.type-dictation { background: rgba(168,85,247,0.14); color: var(--md-violet); }
.type-assistant { background: rgba(251,90,160,0.14); color: var(--md-rose); }
.activity-main { min-width: 0; }
.activity-title { font-size: 14px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.activity-snippet { font-size: 12.5px; color: rgba(var(--v-theme-on-surface),0.4); margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.activity-right { text-align: right; display: flex; flex-direction: column; align-items: flex-end; gap: 5px; }
.activity-time { font-size: 11.5px; color: rgba(var(--v-theme-on-surface),0.4); font-family: 'JetBrains Mono', monospace; }
.doc-chip { font-size: 10.5px; font-weight: 600; color: var(--md-good); background: rgba(52,229,168,0.12); border: 1px solid rgba(52,229,168,0.28); padding: 2px 8px; border-radius: 100px; white-space: nowrap; }
.view-all { display: block; text-align: center; padding: 13px 0 4px; margin-top: 4px; font-size: 12.5px; font-weight: 600; color: rgba(var(--v-theme-on-surface),0.55); text-decoration: none; border-top: 1px solid var(--md-border); transition: color 0.2s ease; }
.view-all:hover { color: var(--md-cyan); }

.donut-wrap { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; padding-bottom: 6px; gap: 16px; }
.donut-figure { position: relative; width: 160px; height: 160px; }
.donut-figure svg { width: 100%; height: 100%; transform: rotate(-90deg); overflow: visible; }
.donut-seg { fill: none; stroke-width: 15; stroke-linecap: round; }
.donut-center { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.donut-center .n { font-family: 'JetBrains Mono', monospace; font-size: 24px; font-weight: 600; }
.donut-center .l { font-size: 10px; letter-spacing: 0.06em; text-transform: uppercase; color: rgba(var(--v-theme-on-surface),0.4); margin-top: 2px; }
.legend { display: flex; flex-direction: column; gap: 9px; width: 100%; }
.legend-row { display: flex; align-items: center; gap: 9px; font-size: 12.5px; }
.legend-dot { width: 9px; height: 9px; border-radius: 3px; flex-shrink: 0; }
.legend-name { color: rgba(var(--v-theme-on-surface),0.55); flex: 1; }
.legend-val { font-family: 'JetBrains Mono', monospace; font-weight: 600; }

@keyframes md-rise { to { opacity: 1; transform: translateY(0) scale(1); } }
@keyframes md-slide { to { opacity: 1; transform: translateX(0); } }

@media (prefers-reduced-motion: reduce) {
  .md-root *, .md-root *::before, .md-root *::after { animation: none !important; }
  .profile-header, .stat-card, .glow-card, .qa-card, .activity-item { opacity: 1 !important; transform: none !important; }
}
@media (max-width: 980px) { .stats-row { grid-template-columns: 1fr 1fr; } .mid-grid { grid-template-columns: 1fr; } .quick-actions { grid-template-columns: 1fr; } .trial-grid { grid-template-columns: 1fr; } }
@media (max-width: 620px) { .stats-row { grid-template-columns: 1fr; } .profile-body { flex-wrap: wrap; } .profile-meta { text-align: left; } }
</style>
