<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">
        <span class="logo">🏨</span>
        <div>
          <h1>HMSMaster</h1>
          <p class="sub">Hotel Management System · Suite Administrativa</p>
        </div>
      </div>
      <div class="suite-badge" title="Facturación delegada a POSMaster">🛒 POSMaster · facturación</div>
      <button v-if="token" class="btn small logout" @click="logout">🔐 Salir</button>
    </header>

    <div v-if="needLogin" class="login-overlay">
      <form class="card login" @submit.prevent="doLogin">
        <h2>🔐 Acceso HMSMaster</h2>
        <p class="hint">La sesión es requerida por el servidor.</p>
        <label>Usuario<input v-model="loginForm.username" required autocomplete="username" /></label>
        <label>Contraseña<input v-model="loginForm.password" type="password" required autocomplete="current-password" /></label>
        <button class="btn primary" type="submit">Entrar</button>
      </form>
    </div>

    <main>
      <section class="stats">
        <div class="card stat"><span class="stat-value">{{ stats.occupancy_pct?.toFixed(0) ?? '—' }}%</span><span class="stat-label">Ocupación</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.rooms ?? '—' }}</span><span class="stat-label">Habitaciones</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.in_house ?? '—' }}</span><span class="stat-label">In-house</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.arrivals_today ?? '—' }}</span><span class="stat-label">Llegadas hoy</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.reservations ?? '—' }}</span><span class="stat-label">Reservas</span></div>
      </section>

      <p v-if="msg" class="toast" :class="msgType">{{ msg }}</p>

      <section class="grid">
        <div class="card">
          <div class="card-head"><h2>Reservas</h2><span class="hint">Acciones por estado</span></div>
          <table>
            <thead><tr><th>Huésped</th><th>Hab.</th><th>Canal</th><th>Entrada</th><th>Salida</th><th>Estado</th><th class="num">Total</th><th>Acción</th></tr></thead>
            <tbody>
              <tr v-for="r in reservations" :key="r.ID">
                <td>{{ r.guest_name }}</td><td>{{ r.room_number || '—' }}</td>
                <td><span class="channel">{{ r.channel }}</span></td>
                <td>{{ fmtDate(r.check_in) }}</td><td>{{ fmtDate(r.check_out) }}</td>
                <td><span class="chip" :class="r.status">{{ statusLabel(r.status) }}</span></td>
                <td class="num">{{ fmtCRC(r.total_amount) }}</td>
                <td class="actions">
                  <button v-if="r.status === 'pending'" class="btn small" @click="changeStatus(r, 'confirmed')">Confirmar</button>
                  <button v-if="r.status === 'confirmed'" class="btn small primary" @click="changeStatus(r, 'checked_in')">Check-in</button>
                  <button v-if="r.status === 'checked_in'" class="btn small primary" @click="changeStatus(r, 'checked_out')">Check-out</button>
                  <button v-if="['pending','confirmed'].includes(r.status)" class="btn small danger" @click="changeStatus(r, 'canceled')">Cancelar</button>
                  <span v-if="['checked_out','canceled'].includes(r.status)" class="hint">—</span>
                </td>
              </tr>
              <tr v-if="!reservations.length"><td colspan="8" class="empty">Sin reservas</td></tr>
            </tbody>
          </table>
        </div>

        <div class="card">
          <h2>Habitaciones · {{ hotelName }}</h2>
          <div class="rooms">
            <div v-for="room in rooms" :key="room.ID" class="room" :class="room.status">
              <span class="room-num">{{ room.number }}</span>
              <span class="room-type">{{ room.type }}</span>
              <span class="room-rate">{{ fmtCRC(room.base_rate) }}</span>
            </div>
          </div>
          <div class="legend">
            <span><i class="dot available"></i>Disponible</span>
            <span><i class="dot occupied"></i>Ocupada</span>
            <span><i class="dot cleaning"></i>Limpieza</span>
            <span><i class="dot maintenance"></i>Mantenimiento</span>
          </div>
        </div>
      </section>

      <section class="grid ops">
        <div class="card">
          <div class="card-head"><h2>Nueva reserva</h2><span class="hint">Tarifa = noches × tarifa base</span></div>
          <form @submit.prevent="createReservation">
            <label>Huésped<input v-model="form.guest_name" required placeholder="Nombre completo" /></label>
            <label>Email<input v-model="form.guest_email" type="email" placeholder="opcional" /></label>
            <label>Habitación
              <select v-model.number="form.room_id" required>
                <option v-for="r in rooms" :key="r.ID" :value="r.ID">{{ r.number }} · {{ r.type }} · {{ fmtCRC(r.base_rate) }}</option>
              </select>
            </label>
            <div class="row">
              <label>Entrada<input v-model="form.check_in" type="date" required /></label>
              <label>Salida<input v-model="form.check_out" type="date" required /></label>
            </div>
            <label>Canal
              <select v-model="form.channel">
                <option value="direct">Directo</option><option value="airbnb">Airbnb</option>
                <option value="booking">Booking</option><option value="expedia">Expedia</option>
              </select>
            </label>
            <button class="btn primary" type="submit">Crear reserva</button>
          </form>
        </div>

        <div class="card">
          <div class="card-head"><h2>Folio · POSMaster</h2><span class="hint">Cargos → /api/facturas</span></div>
          <label>Reserva
            <select v-model.number="folioReservationId" @change="loadCharges">
              <option v-for="r in reservations" :key="r.ID" :value="r.ID">#{{ r.ID }} · {{ r.guest_name }} · {{ statusLabel(r.status) }}</option>
            </select>
          </label>
          <div class="charges">
            <div v-for="c in charges" :key="c.ID" class="charge">
              <span>{{ c.description }}</span>
              <span class="charge-amount">{{ fmtCRC(c.amount) }}</span>
              <span class="chip sync" :class="c.sync_status">{{ syncLabel(c.sync_status) }}</span>
            </div>
            <p v-if="!charges.length" class="hint">Sin cargos en esta reserva.</p>
          </div>
          <form @submit.prevent="addCharge">
            <label>Descripción<input v-model="charge.description" required placeholder="Ej. Minibar, lavandería" /></label>
            <label>Monto (₡)<input v-model.number="charge.amount" type="number" min="1" step="1" required /></label>
            <button class="btn primary" type="submit">Agregar cargo</button>
          </form>
          <p class="hint small">Sin POSMASTER_API_URL los cargos quedan locales. Con el token configurado se envían como línea de factura.</p>
        </div>
      </section>
    </main>

    <footer>HMSMaster v0.2.0 · parte de la suite con 🛒 POSMaster · datos demo en CRC</footer>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'

const stats = ref({})
const reservations = ref([])
const rooms = ref([])
const hotelName = ref('')
const charges = ref([])
const msg = ref('')
const msgType = ref('ok')
const folioReservationId = ref(null)
const form = reactive({ guest_name: '', guest_email: '', room_id: 0, check_in: '', check_out: '', channel: 'direct' })
const charge = reactive({ description: '', amount: 0 })
let timer = null

const token = ref(localStorage.getItem('hms_token') || '')
const needLogin = ref(false)
const loginForm = reactive({ username: '', password: '' })

async function apiFetch(url, options = {}) {
  const headers = Object.assign({}, options.headers || {})
  if (token.value) headers['Authorization'] = 'Bearer ' + token.value
  const res = await fetch(url, Object.assign({}, options, { headers }))
  if (res.status === 401) {
    needLogin.value = true
    token.value = ''
    localStorage.removeItem('hms_token')
    throw new Error('Sesión expirada')
  }
  return res
}

async function doLogin() {
  try {
    const res = await apiFetch('/api/auth/login', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(loginForm)
    })
    if (!res.ok) throw new Error('Credenciales inválidas')
    const data = await res.json()
    token.value = data.token
    localStorage.setItem('hms_token', data.token)
    loginForm.password = ''
    needLogin.value = false
    notify('Sesión iniciada')
    await load()
  } catch (e) { notify('Login: ' + e.message, 'err') }
}

function logout() {
  token.value = ''
  localStorage.removeItem('hms_token')
  needLogin.value = true
}

const crc = new Intl.NumberFormat('es-CR', { style: 'currency', currency: 'CRC', maximumFractionDigits: 0 })
const fmtCRC = (v) => crc.format(v || 0)
const fmtDate = (v) => v ? new Date(v).toLocaleDateString('es-CR', { day: '2-digit', month: 'short' }) : '—'
const statusLabel = (s) => ({ pending: 'Pendiente', confirmed: 'Confirmada', checked_in: 'Check-in', checked_out: 'Check-out', canceled: 'Cancelada' }[s] || s)
const syncLabel = (s) => ({ local: 'Local', synced: 'Facturado', failed: 'Error sync' }[s] || s)
const notify = (text, type = 'ok') => { msg.value = text; msgType.value = type; setTimeout(() => { msg.value = '' }, 5000) }

async function loadCharges() {
  if (!folioReservationId.value) { charges.value = []; return }
  try { charges.value = await apiFetch(`/api/reservations/${folioReservationId.value}/charges`).then(x => x.json()) }
  catch (e) { console.error(e) }
}

async function load() {
  try {
    const [s, r, h] = await Promise.all([
      apiFetch('/api/stats').then(x => x.json()),
      apiFetch('/api/reservations').then(x => x.json()),
      apiFetch('/api/hotels').then(x => x.json()),
    ])
    stats.value = s
    reservations.value = r
    if (h.length) {
      hotelName.value = h[0].name
      rooms.value = await apiFetch(`/api/hotels/${h[0].ID}/rooms`).then(x => x.json())
    }
    if (!folioReservationId.value && r.length) folioReservationId.value = r[0].ID
    await loadCharges()
  } catch (e) { console.error('API no disponible', e) }
}

async function changeStatus(r, status) {
  try {
    const res = await apiFetch(`/api/reservations/${r.ID}/status`, {
      method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ status })
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    notify(`Reserva #${r.ID}: ${statusLabel(status)}`)
    await load()
  } catch (e) { notify('No se pudo cambiar el estado: ' + e.message, 'err') }
}

async function createReservation() {
  const payload = {
    hotel_id: 1, room_id: form.room_id, guest_name: form.guest_name, guest_email: form.guest_email,
    check_in: form.check_in + 'T14:00:00Z', check_out: form.check_out + 'T12:00:00Z', channel: form.channel
  }
  try {
    const res = await apiFetch('/api/reservations', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload)
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    const created = await res.json()
    notify(`Reserva #${created.ID} creada por ${fmtCRC(created.total_amount)}`)
    form.guest_name = ''; form.guest_email = ''
    folioReservationId.value = created.ID
    await load()
  } catch (e) { notify('No se pudo crear: ' + e.message, 'err') }
}

async function addCharge() {
  try {
    const res = await apiFetch(`/api/reservations/${folioReservationId.value}/charges`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ description: charge.description, amount: Number(charge.amount) })
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    notify('Cargo agregado al folio')
    charge.description = ''; charge.amount = 0
    await loadCharges()
  } catch (e) { notify('No se pudo agregar el cargo: ' + e.message, 'err') }
}

onMounted(() => { load(); timer = setInterval(load, 30000) })
onUnmounted(() => clearInterval(timer))
</script>

<style>
:root {
  --bg: #0f172a; --surface: #1e293b; --border: #334155;
  --text: #e2e8f0; --muted: #94a3b8; --primary: #818cf8; --accent: #6366f1;
}
* { box-sizing: border-box; margin: 0; }
body { background: var(--bg); color: var(--text); font-family: 'Inter', system-ui, sans-serif; }
.app { min-height: 100vh; display: flex; flex-direction: column; }
.topbar { display: flex; justify-content: space-between; align-items: center; padding: 16px 28px; border-bottom: 1px solid var(--border); background: var(--surface); }
.brand { display: flex; gap: 12px; align-items: center; }
.logo { font-size: 30px; }
h1 { font-size: 20px; letter-spacing: .3px; }
.sub { color: var(--muted); font-size: 12px; }
.suite-badge { background: rgba(99,102,241,.15); color: var(--primary); border: 1px solid var(--accent); border-radius: 999px; padding: 6px 14px; font-size: 13px; }
main { padding: 24px 28px; flex: 1; }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 14px; margin-bottom: 20px; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 18px; }
.stat { display: flex; flex-direction: column; gap: 4px; }
.stat-value { font-size: 28px; font-weight: 700; color: var(--primary); }
.stat-label { color: var(--muted); font-size: 13px; }
.grid { display: grid; grid-template-columns: 3fr 2fr; gap: 14px; margin-bottom: 14px; }
@media (max-width: 980px) { .grid { grid-template-columns: 1fr; } }
.card-head { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 12px; }
h2 { font-size: 15px; color: var(--text); }
.hint { color: var(--muted); font-size: 11px; }
.hint.small { display: block; margin-top: 10px; line-height: 1.4; }
table { width: 100%; border-collapse: collapse; font-size: 12px; }
th { text-align: left; color: var(--muted); font-weight: 500; padding: 6px 6px; border-bottom: 1px solid var(--border); }
td { padding: 7px 6px; border-bottom: 1px solid rgba(51,65,85,.5); vertical-align: middle; }
.num { text-align: right; }
.empty { color: var(--muted); text-align: center; }
.channel { color: var(--muted); font-size: 11px; }
.chip { border-radius: 999px; padding: 3px 9px; font-size: 11px; font-weight: 600; }
.chip.pending { background: rgba(251,191,36,.15); color: #fbbf24; }
.chip.confirmed { background: rgba(96,165,250,.15); color: #60a5fa; }
.chip.checked_in { background: rgba(52,211,153,.15); color: #34d399; }
.chip.checked_out { background: rgba(148,163,184,.15); color: #94a3b8; }
.chip.canceled { background: rgba(248,113,113,.15); color: #f87171; }
.chip.sync { margin-left: auto; }
.chip.local { background: rgba(148,163,184,.15); color: #94a3b8; }
.chip.synced { background: rgba(52,211,153,.15); color: #34d399; }
.chip.failed { background: rgba(248,113,113,.15); color: #f87171; }
.actions { white-space: nowrap; }
.btn { background: var(--accent); color: white; border: 1px solid var(--accent); border-radius: 7px; padding: 8px 14px; font-size: 12px; font-weight: 600; cursor: pointer; }
.btn:hover { filter: brightness(1.15); }
.btn.small { padding: 4px 8px; font-size: 11px; margin-right: 4px; background: transparent; color: var(--muted); border-color: var(--border); }
.btn.small.primary { color: var(--primary); border-color: var(--accent); background: rgba(99,102,241,.15); }
.btn.small.danger { color: #f87171; border-color: rgba(248,113,113,.4); }
form { display: flex; flex-direction: column; gap: 10px; }
label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--muted); }
input, select { background: #0f172a; color: var(--text); border: 1px solid var(--border); border-radius: 7px; padding: 8px 10px; font-size: 13px; font-family: inherit; }
input:focus, select:focus { outline: 1px solid var(--accent); }
.row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.rooms { display: grid; grid-template-columns: repeat(auto-fill, minmax(95px,1fr)); gap: 10px; }
.room { border-radius: 10px; padding: 10px; display: flex; flex-direction: column; gap: 2px; border: 1px solid var(--border); background: rgba(15,23,42,.6); }
.room-num { font-weight: 700; font-size: 16px; }
.room-type { color: var(--muted); font-size: 11px; text-transform: capitalize; }
.room-rate { font-size: 11px; color: var(--muted); }
.room.available { border-color: #34d399; } .room.occupied { border-color: #f87171; }
.room.cleaning { border-color: #fbbf24; } .room.maintenance { border-color: #64748b; opacity: .7; }
.legend { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 14px; color: var(--muted); font-size: 12px; }
.dot { display: inline-block; width: 9px; height: 9px; border-radius: 50%; margin-right: 5px; }
.dot.available { background: #34d399; } .dot.occupied { background: #f87171; }
.dot.cleaning { background: #fbbf24; } .dot.maintenance { background: #64748b; }
.charges { display: flex; flex-direction: column; gap: 6px; margin: 12px 0; }
.charge { display: flex; align-items: center; gap: 8px; font-size: 12px; background: rgba(15,23,42,.6); border: 1px solid var(--border); border-radius: 7px; padding: 7px 10px; }
.charge-amount { color: var(--primary); font-weight: 600; }
.toast { margin-bottom: 14px; padding: 10px 14px; border-radius: 8px; font-size: 13px; }
.toast.ok { background: rgba(52,211,153,.15); color: #34d399; border: 1px solid rgba(52,211,153,.4); }
.toast.err { background: rgba(248,113,113,.15); color: #f87171; border: 1px solid rgba(248,113,113,.4); }
footer { padding: 14px 28px; color: var(--muted); font-size: 12px; border-top: 1px solid var(--border); }
.login-overlay { position: fixed; inset: 0; background: rgba(15,23,42,.88); display: flex; align-items: center; justify-content: center; z-index: 50; }
.login { width: 340px; display: flex; flex-direction: column; gap: 12px; box-shadow: 0 20px 60px rgba(0,0,0,.5); }
.login h2 { margin-bottom: 4px; }

</style>
