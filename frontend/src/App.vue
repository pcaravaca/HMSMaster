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
      <div class="suite-badge" title="Facturación delegada a POSMaster">
        🛒 POSMaster · facturación
      </div>
    </header>

    <main>
      <section class="stats">
        <div class="card stat">
          <span class="stat-value">{{ stats.occupancy_pct?.toFixed(0) ?? '—' }}%</span>
          <span class="stat-label">Ocupación</span>
        </div>
        <div class="card stat">
          <span class="stat-value">{{ stats.rooms ?? '—' }}</span>
          <span class="stat-label">Habitaciones</span>
        </div>
        <div class="card stat">
          <span class="stat-value">{{ stats.in_house ?? '—' }}</span>
          <span class="stat-label">In-house</span>
        </div>
        <div class="card stat">
          <span class="stat-value">{{ stats.arrivals_today ?? '—' }}</span>
          <span class="stat-label">Llegadas hoy</span>
        </div>
        <div class="card stat">
          <span class="stat-value">{{ stats.reservations ?? '—' }}</span>
          <span class="stat-label">Reservas</span>
        </div>
      </section>

      <section class="grid">
        <div class="card">
          <h2>Reservas</h2>
          <table>
            <thead>
              <tr>
                <th>Huésped</th><th>Hab.</th><th>Canal</th>
                <th>Entrada</th><th>Salida</th><th>Estado</th><th class="num">Total</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in reservations" :key="r.ID">
                <td>{{ r.guest_name }}</td>
                <td>{{ r.room_number || '—' }}</td>
                <td><span class="channel">{{ r.channel }}</span></td>
                <td>{{ fmtDate(r.check_in) }}</td>
                <td>{{ fmtDate(r.check_out) }}</td>
                <td><span class="chip" :class="r.status">{{ statusLabel(r.status) }}</span></td>
                <td class="num">{{ fmtCRC(r.total_amount) }}</td>
              </tr>
              <tr v-if="!reservations.length"><td colspan="7" class="empty">Sin reservas</td></tr>
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
    </main>

    <footer>
      HMSMaster v0.1.0 · parte de la suite con 🛒 POSMaster · datos demo en CRC
    </footer>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const stats = ref({})
const reservations = ref([])
const rooms = ref([])
const hotelName = ref('')
let timer = null

const crc = new Intl.NumberFormat('es-CR', { style: 'currency', currency: 'CRC', maximumFractionDigits: 0 })
const fmtCRC = (v) => crc.format(v || 0)
const fmtDate = (v) => v ? new Date(v).toLocaleDateString('es-CR', { day: '2-digit', month: 'short' }) : '—'
const statusLabel = (s) => ({
  pending: 'Pendiente', confirmed: 'Confirmada', checked_in: 'Check-in',
  checked_out: 'Check-out', canceled: 'Cancelada',
}[s] || s)

async function load() {
  try {
    const [s, r, h] = await Promise.all([
      fetch('/api/stats').then(x => x.json()),
      fetch('/api/reservations').then(x => x.json()),
      fetch('/api/hotels').then(x => x.json()),
    ])
    stats.value = s
    reservations.value = r
    if (h.length) {
      hotelName.value = h[0].name
      rooms.value = await fetch(`/api/hotels/${h[0].ID}/rooms`).then(x => x.json())
    }
  } catch (e) {
    console.error('API no disponible', e)
  }
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
.topbar {
  display: flex; justify-content: space-between; align-items: center;
  padding: 16px 28px; border-bottom: 1px solid var(--border); background: var(--surface);
}
.brand { display: flex; gap: 12px; align-items: center; }
.logo { font-size: 30px; }
h1 { font-size: 20px; letter-spacing: .3px; }
.sub { color: var(--muted); font-size: 12px; }
.suite-badge {
  background: rgba(99, 102, 241, .15); color: var(--primary);
  border: 1px solid var(--accent); border-radius: 999px; padding: 6px 14px; font-size: 13px;
}
main { padding: 24px 28px; flex: 1; }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 14px; margin-bottom: 20px; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 18px; }
.stat { display: flex; flex-direction: column; gap: 4px; }
.stat-value { font-size: 28px; font-weight: 700; color: var(--primary); }
.stat-label { color: var(--muted); font-size: 13px; }
.grid { display: grid; grid-template-columns: 3fr 2fr; gap: 14px; }
@media (max-width: 980px) { .grid { grid-template-columns: 1fr; } }
h2 { font-size: 15px; margin-bottom: 12px; color: var(--text); }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; color: var(--muted); font-weight: 500; padding: 6px 8px; border-bottom: 1px solid var(--border); }
td { padding: 8px; border-bottom: 1px solid rgba(51, 65, 85, .5); }
.num { text-align: right; }
.empty { color: var(--muted); text-align: center; }
.channel { color: var(--muted); font-size: 12px; }
.chip { border-radius: 999px; padding: 3px 10px; font-size: 11px; font-weight: 600; }
.chip.pending { background: rgba(251, 191, 36, .15); color: #fbbf24; }
.chip.confirmed { background: rgba(96, 165, 250, .15); color: #60a5fa; }
.chip.checked_in { background: rgba(52, 211, 153, .15); color: #34d399; }
.chip.checked_out { background: rgba(148, 163, 184, .15); color: #94a3b8; }
.chip.canceled { background: rgba(248, 113, 113, .15); color: #f87171; }
.rooms { display: grid; grid-template-columns: repeat(auto-fill, minmax(95px, 1fr)); gap: 10px; }
.room {
  border-radius: 10px; padding: 10px; display: flex; flex-direction: column; gap: 2px;
  border: 1px solid var(--border); background: rgba(15, 23, 42, .6);
}
.room-num { font-weight: 700; font-size: 16px; }
.room-type { color: var(--muted); font-size: 11px; text-transform: capitalize; }
.room-rate { font-size: 11px; color: var(--muted); }
.room.available { border-color: #34d399; }
.room.occupied { border-color: #f87171; }
.room.cleaning { border-color: #fbbf24; }
.room.maintenance { border-color: #64748b; opacity: .7; }
.legend { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 14px; color: var(--muted); font-size: 12px; }
.dot { display: inline-block; width: 9px; height: 9px; border-radius: 50%; margin-right: 5px; }
.dot.available { background: #34d399; } .dot.occupied { background: #f87171; }
.dot.cleaning { background: #fbbf24; } .dot.maintenance { background: #64748b; }
footer { padding: 14px 28px; color: var(--muted); font-size: 12px; border-top: 1px solid var(--border); }
</style>
