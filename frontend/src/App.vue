<template>
  <div class="app">
    <aside class="sidebar">
      <div class="side-brand"><span class="logo">🏨</span><div><strong>HMSMaster</strong><small>Suite Administrativa</small></div></div>
      <nav class="side-nav">
        <div class="nav-group">Operación</div>
        <button :class="{ active: activeView === 'panel' }" @click="activeView = 'panel'">▦ Panel</button>
        <button :class="{ active: activeView === 'reservas' }" @click="activeView = 'reservas'">▤ Reservas</button>
        <button :class="{ active: activeView === 'habitaciones' }" @click="activeView = 'habitaciones'">▥ Habitaciones</button>
        <button :class="{ active: activeView === 'huespedes' }" @click="activeView = 'huespedes'">◉ Huéspedes</button>
        <button :class="{ active: activeView === 'folio' }" @click="activeView = 'folio'">▧ Folio · POSMaster</button>
        <div class="nav-group">Configuración</div>
        <button :class="{ active: activeView === 'propiedades' }" @click="activeView = 'propiedades'">⬡ Propiedades</button>
        <button :class="{ active: activeView === 'tarifas' }" @click="activeView = 'tarifas'">◇ Tarifas dinámicas</button>
        <button :class="{ active: activeView === 'canales' }" @click="activeView = 'canales'">⇄ Canales OTA</button>
        <button :class="{ active: activeView === 'limpieza' }" @click="activeView = 'limpieza'">✦ Limpieza</button>
        <button :class="{ active: activeView === 'reportes' }" @click="activeView = 'reportes'">▨ Reportes</button>
        <button :class="{ active: activeView === 'usuarios' }" @click="activeView = 'usuarios'">◈ Usuarios y roles</button>
      </nav>
      <div class="side-foot">v0.15.0 · navegación por módulos</div>
    </aside>
    <div class="workspace">
    <header class="topbar">
      <div class="brand">
        <span class="logo">🏨</span>
        <div>
          <h1>HMSMaster</h1>
          <p class="sub">Hotel Management System · Suite Administrativa</p>
        </div>
      </div>
      <span class="page-title">{{ pageTitle }}</span>
      <div class="suite-badge" title="Facturación delegada a POSMaster">🛒 POSMaster · facturación</div>
      <select v-model.number="activeHotelId" @change="onHotelChange" class="hotel-select" title="Propiedad activa">
        <option v-for="h in hotels" :key="h.ID" :value="h.ID">{{ h.name }}</option>
      </select>
      <span v-if="token" class="chip role" :title="me.username">{{ me.role }}</span>
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
      <section class="stats" v-if="activeView === 'panel'">
        <div class="card stat"><span class="stat-value">{{ stats.occupancy_pct?.toFixed(0) ?? '—' }}%</span><span class="stat-label">Ocupación</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.rooms ?? '—' }}</span><span class="stat-label">Habitaciones</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.in_house ?? '—' }}</span><span class="stat-label">In-house</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.arrivals_today ?? '—' }}</span><span class="stat-label">Llegadas hoy</span></div>
        <div class="card stat"><span class="stat-value">{{ stats.reservations ?? '—' }}</span><span class="stat-label">Reservas</span></div>
      </section>

      <p v-if="msg" class="toast" :class="msgType">{{ msg }}</p>

      <section class="grid">
        <div class="card" v-if="activeView === 'reservas'">
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
                  <button v-if="canOps && r.status === 'pending'" class="btn small" @click="changeStatus(r, 'confirmed')">Confirmar</button>
                  <button v-if="canOps && r.status === 'confirmed'" class="btn small primary" @click="changeStatus(r, 'checked_in')">Check-in</button>
                  <button v-if="canOps && r.status === 'checked_in'" class="btn small primary" @click="changeStatus(r, 'checked_out')">Check-out</button>
                  <button v-if="canOps && ['pending','confirmed'].includes(r.status)" class="btn small danger" @click="changeStatus(r, 'canceled')">Cancelar</button>
                  <span v-if="['checked_out','canceled'].includes(r.status)" class="hint">—</span>
                </td>
              </tr>
              <tr v-if="!reservations.length"><td colspan="8" class="empty">Sin reservas</td></tr>
            </tbody>
          </table>
        </div>

        <div class="card" v-if="activeView === 'habitaciones'">
          <h2>Habitaciones · {{ hotelName }}</h2>
          <div class="avail-search">
            <input v-model="search.check_in" type="date" />
            <input v-model="search.check_out" type="date" />
            <button class="btn small primary" @click="searchAvailability">Buscar</button>
          </div>
          <p v-if="availabilitySummary" class="hint small">{{ availabilitySummary }}</p>
          <div class="rooms">
            <div v-for="room in displayRooms" :key="room.ID" class="room" :class="[room.status, availability && !room.available ? 'blocked' : '']">
              <span class="room-num">{{ room.number }}</span>
              <span class="room-type">{{ room.type }}</span>
              <span class="room-rate">{{ fmtCRC(room.base_rate) }}</span>
              <span v-if="availability && !room.available" class="room-blocked">No libre en fechas</span>
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
        <div class="card" v-if="activeView === 'reservas'">
          <div class="card-head"><h2>Nueva reserva</h2><span class="hint">{{ quoteText }}</span></div>
          <form @submit.prevent="createReservation">
            <label>Huésped registrado
              <select v-model.number="form.guest_id" @change="onGuestSelect">
                <option :value="0">— Nuevo huésped —</option>
                <option v-for="g in guests" :key="g.ID" :value="g.ID">{{ g.first_name }} {{ g.last_name }} · {{ g.document_number }}</option>
              </select>
            </label>
            <label>Huésped<input v-model="form.guest_name" required placeholder="Nombre completo" /></label>
            <label>Email<input v-model="form.guest_email" type="email" placeholder="opcional" /></label>
            <label>Habitación
              <select v-model.number="form.room_id" required @change="loadQuote">
                <option v-for="r in visibleRooms" :key="r.ID" :value="r.ID">{{ r.number }} · {{ r.type }} · {{ fmtCRC(r.base_rate) }}</option>
              </select>
            </label>
            <div class="row">
              <label>Entrada<input v-model="form.check_in" type="date" required @change="loadQuote" /></label>
              <label>Salida<input v-model="form.check_out" type="date" required @change="loadQuote" /></label>
            </div>
            <label>Canal
              <select v-model="form.channel">
                <option value="direct">Directo</option><option value="airbnb">Airbnb</option>
                <option value="booking">Booking</option><option value="expedia">Expedia</option>
              </select>
            </label>
            <button v-if="canOps" class="btn primary" type="submit">Crear reserva</button>
          </form>
        </div>

        <div class="card" v-if="activeView === 'folio'">
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
              <span class="chip sync" :class="c.sync_status" :title="c.sync_error || ''">{{ syncLabel(c.sync_status) }}</span>
              <button v-if="canBilling && ['failed','uncertain'].includes(c.sync_status)" class="btn small" @click="retryCharge(c)">Reintentar</button>
            </div>
            <p v-if="!charges.length" class="hint">Sin cargos en esta reserva.</p>
          </div>
          <form @submit.prevent="addCharge">
            <label>Descripción<input v-model="charge.description" required placeholder="Ej. Minibar, lavandería" /></label>
            <label>Monto (₡)<input v-model.number="charge.amount" type="number" min="1" step="1" required /></label>
            <button v-if="canBilling" class="btn primary" type="submit">Agregar cargo</button>
          </form>
          <p class="hint small">Sin POSMASTER_API_URL los cargos quedan locales. Con el token configurado se envían como línea de factura.</p>
        </div>

        <div class="card" v-if="activeView === 'reportes'">
          <div class="card-head"><h2>Reportes</h2><span class="hint">Ocupación · ADR · RevPAR · canales</span></div>
          <form @submit.prevent="loadReport">
            <div class="row">
              <label>Desde<input v-model="reportForm.from" type="date" required /></label>
              <label>Hasta<input v-model="reportForm.to" type="date" required /></label>
            </div>
            <button class="btn primary" type="submit">Generar reporte</button>
          </form>
          <div v-if="report" class="stats mini">
            <div class="card stat"><span class="stat-value">{{ report.occupancy_pct }}%</span><span class="stat-label">Ocupación</span></div>
            <div class="card stat"><span class="stat-value">{{ fmtCRC(report.adr) }}</span><span class="stat-label">ADR</span></div>
            <div class="card stat"><span class="stat-value">{{ fmtCRC(report.revpar) }}</span><span class="stat-label">RevPAR</span></div>
            <div class="card stat"><span class="stat-value">{{ fmtCRC(report.room_revenue) }}</span><span class="stat-label">Ingreso</span></div>
          </div>
          <div v-if="report" class="charges">
            <div v-for="ch in report.by_channel" :key="ch.channel" class="charge">
              <span>{{ ch.channel }}</span><span class="hint">{{ ch.reservations }} reservas · {{ fmtCRC(ch.revenue) }}</span>
            </div>
          </div>
          <table v-if="report" class="report-table">
            <thead><tr><th>Fecha</th><th>Noches</th><th>Ingreso</th></tr></thead>
            <tbody><tr v-for="d in report.by_day" :key="d.date"><td>{{ d.date }}</td><td>{{ d.occupied }}</td><td class="num">{{ fmtCRC(d.revenue) }}</td></tr></tbody>
          </table>
          <p v-if="!report" class="hint">Elegí un rango y generá el reporte.</p>
        </div>

        <div class="card" v-if="activeView === 'limpieza'">
          <div class="card-head"><h2>Limpieza</h2><span class="hint">{{ hkTasks.filter(t => t.status !== 'done').length }} tareas abiertas</span></div>
          <label>Ver
            <select v-model="hkFilter">
              <option value="open">Abiertas</option><option value="pending">Pendientes</option>
              <option value="in_progress">En curso</option><option value="done">Listas</option>
              <option value="all">Todas</option>
            </select>
          </label>
          <form v-if="canClean" @submit.prevent="createHkTask">
            <label>Habitación
              <select v-model.number="hkForm.room_id">
                <option v-for="r in rooms" :key="r.ID" :value="r.ID">{{ r.number }} · {{ r.status }}</option>
              </select>
            </label>
            <label>Tipo
              <select v-model="hkForm.type">
                <option value="checkout_clean">Limpieza de salida</option>
                <option value="stayover">Limpieza en estancia</option>
                <option value="maintenance">Mantenimiento</option>
              </select>
            </label>
            <label>Nota<input v-model="hkForm.notes" placeholder="Ej. revisar minibar" /></label>
            <button class="btn primary" type="submit">Crear tarea</button>
          </form>
          <div class="charges">
            <div v-for="t in filteredHkTasks" :key="t.ID" class="charge">
              <span><b>Hab. {{ t.room_number }}</b> · {{ hkTypeLabel(t.type) }}</span>
              <span class="hint">{{ t.assigned_to || 'sin asignar' }}{{ t.notes ? ' · ' + t.notes : '' }}</span>
              <span class="chip" :class="t.status">{{ hkStatusLabel(t.status) }}</span>
              <button v-if="canClean && t.status === 'pending'" class="btn small" @click="updateHkTask(t, 'in_progress')">Tomar</button>
              <button v-if="canClean && t.status !== 'done'" class="btn small primary" @click="updateHkTask(t, 'done')">Completar</button>
            </div>
            <p v-if="!filteredHkTasks.length" class="hint">Sin tareas en esta vista.</p>
          </div>
          <p class="hint small">El check-out crea una tarea y pone la habitación en limpieza; al completarla vuelve a disponible.</p>
        </div>

        <div class="card" v-if="activeView === 'canales'">
          <div class="card-head"><h2>Canales OTA</h2><span class="hint">{{ channelBookings.length }} reservas entrantes</span></div>
          <form v-if="canOps" @submit.prevent="createChannelBooking">
            <label>Canal
              <select v-model="channelForm.channel">
                <option value="airbnb">Airbnb</option><option value="booking">Booking</option>
                <option value="expedia">Expedia</option><option value="direct">Directo</option>
              </select>
            </label>
            <label>Referencia externa<input v-model="channelForm.external_id" required placeholder="Ej. AB-12345" /></label>
            <label>Huésped<input v-model="channelForm.guest_name" required /></label>
            <label>Habitación
              <select v-model.number="channelForm.room_id">
                <option v-for="r in rooms" :key="r.ID" :value="r.ID">{{ r.number }} · {{ r.type }}</option>
              </select>
            </label>
            <div class="row">
              <label>Entrada<input v-model="channelForm.check_in" type="date" required /></label>
              <label>Salida<input v-model="channelForm.check_out" type="date" required /></label>
            </div>
            <button class="btn primary" type="submit">Importar reserva</button>
          </form>
          <div class="charges">
            <div v-for="b in channelBookings" :key="b.ID" class="charge">
              <span><b>{{ b.channel }}</b> · {{ b.external_id }}</span>
              <span class="hint">{{ b.guest_name }} · {{ fmtDate(b.check_in) }} → {{ fmtDate(b.check_out) }}</span>
              <span class="chip" :class="b.status">{{ b.status }}</span>
              <button v-if="canOps && b.status !== 'canceled'" class="btn small" @click="cancelChannelBooking(b)">Cancelar</button>
            </div>
            <p v-if="!channelBookings.length" class="hint">Sin reservas entrantes.</p>
          </div>
          <p class="hint small">Webhook real: POST /api/channels/bookings con cabecera X-HMS-Signature (HMAC-SHA256) cuando el canal tiene secreto configurado.
          La repetición de la misma referencia externa no crea una reserva duplicada.</p>
        </div>

        <div class="card" v-if="activeView === 'usuarios' && canAdmin">
          <div class="card-head"><h2>Usuarios y roles</h2><span class="hint">{{ users.length }} cuentas</span></div>
          <div class="charges">
            <div v-for="u in users" :key="u.ID" class="charge">
              <span>{{ u.username }}</span><span class="hint">{{ u.role }} · {{ u.active ? 'activo' : 'inactivo' }}</span>
              <button v-if="u.username !== me.username" class="btn small" @click="toggleUser(u)">{{ u.active ? 'Desactivar' : 'Activar' }}</button>
            </div>
          </div>
          <form @submit.prevent="createUser">
            <label>Usuario<input v-model="userForm.username" required minlength="3" /></label>
            <label>Contraseña<input v-model="userForm.password" type="password" minlength="8" required /></label>
            <label>Rol
              <select v-model="userForm.role">
                <option value="admin">Admin</option><option value="manager">Gerente</option>
                <option value="reception">Recepción</option><option value="housekeeping">Limpieza</option>
                <option value="accounting">Contabilidad</option>
              </select>
            </label>
            <button class="btn primary" type="submit">Agregar usuario</button>
          </form>
          <p class="hint small">Las contraseñas se guardan como hash bcrypt; solo el admin ve esta tarjeta.</p>
        </div>

        <div class="card" v-if="activeView === 'tarifas'">
          <div class="card-head"><h2>Tarifas dinámicas</h2><span class="hint">{{ rateRules.length }} reglas · {{ hotelName }}</span></div>
          <form v-if="canRates" @submit.prevent="createRate">
            <label>Aplica a
              <select v-model.number="rateForm.room_id">
                <option :value="0">Toda la propiedad</option>
                <option v-for="r in rooms" :key="r.ID" :value="r.ID">{{ r.number }} · {{ r.type }} · base {{ fmtCRC(r.base_rate) }}</option>
              </select>
            </label>
            <div class="row">
              <label>Desde<input v-model="rateForm.start_date" type="date" required /></label>
              <label>Hasta<input v-model="rateForm.end_date" type="date" required /></label>
            </div>
            <label>Precio por noche (₡)<input v-model.number="rateForm.price" type="number" min="1" step="1" required /></label>
            <label>Nota<input v-model="rateForm.notes" placeholder="Ej. temporada alta" /></label>
            <button class="btn primary" type="submit">Agregar tarifa</button>
          </form>
          <div class="charges">
            <div v-for="r in rateRules" :key="r.ID" class="charge">
              <span>{{ r.room_id ? 'Hab. ' + (rooms.find(x => x.ID === r.room_id)?.number || r.room_id) : 'Toda la propiedad' }}</span>
              <span class="hint">{{ r.start_date }} → {{ r.end_date }} · {{ fmtCRC(r.price) }}</span>
              <button v-if="canRates" class="btn small" @click="deleteRate(r)">Eliminar</button>
            </div>
            <p v-if="!rateRules.length" class="hint">Sin reglas: se usa la tarifa base de cada habitación.</p>
          </div>
          <p class="hint small">Una regla por habitación gana sobre la de propiedad; entre reglas iguales gana la más reciente.</p>
        </div>

        <div class="card" v-if="activeView === 'propiedades'">
          <div class="card-head"><h2>Propiedades</h2><span class="hint">{{ hotels.length }} en la suite</span></div>
          <div class="charges">
            <div v-for="h in hotels" :key="h.ID" class="charge">
              <span>{{ h.name }}</span><span class="hint">{{ h.address }}</span>
            </div>
          </div>
          <form v-if="canRates" @submit.prevent="createProperty">
            <label>Nombre<input v-model="propertyForm.name" required placeholder="Nueva propiedad" /></label>
            <label>Dirección<input v-model="propertyForm.address" placeholder="Ciudad, país" /></label>
            <button class="btn primary" type="submit">Agregar propiedad</button>
          </form>
        </div>

        <div class="card" v-if="activeView === 'huespedes'">
          <div class="card-head"><h2>Huéspedes</h2><span class="hint">{{ guests.length }} perfiles</span></div>
          <label>Buscar<input v-model="guestSearch" placeholder="Nombre, email o documento" /></label>
          <div class="charges">
            <div v-for="g in guests.filter(x => !guestSearch || [x.first_name, x.last_name, x.email, x.document_number].join(' ').toLowerCase().includes(guestSearch.toLowerCase()))" :key="g.ID" class="charge">
              <span>{{ g.first_name }} {{ g.last_name }}</span>
              <span class="hint">{{ g.document_number }}</span>
              <button class="btn small" @click="viewGuest(g)">Historial</button>
            </div>
            <p v-if="!guests.length" class="hint">Sin huéspedes registrados.</p>
          </div>
          <p v-if="guestDetail" class="hint small">{{ guestDetail.guest.first_name }} {{ guestDetail.guest.last_name }} · {{ guestDetail.reservations.length }} estancia(s) · {{ guestDetail.guest.email }}</p>
          <form @submit.prevent="createGuest">
            <label>Nombre<input v-model="guestForm.first_name" required placeholder="Nombre" /></label>
            <label>Apellido<input v-model="guestForm.last_name" placeholder="Apellido" /></label>
            <label>Email<input v-model="guestForm.email" type="email" placeholder="opcional" /></label>
            <label>Documento
              <div class="row">
                <select v-model="guestForm.document_type"><option value="cedula">Cédula</option><option value="pasaporte">Pasaporte</option><option value="dimex">DIMEX</option></select>
                <input v-model="guestForm.document_number" placeholder="Número" />
              </div>
            </label>
            <button class="btn primary" type="submit">Agregar huésped</button>
          </form>
        </div>
      </section>
    </main>

    <footer>HMSMaster v0.15.0 · parte de la suite con 🛒 POSMaster · datos demo en CRC</footer>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'

const stats = ref({})
const reservations = ref([])
const rooms = ref([])
const availability = ref(null)
const search = reactive({ check_in: '', check_out: '' })
const displayRooms = computed(() => availability.value ? availability.value.rooms : rooms.value)
const visibleRooms = computed(() => displayRooms.value.filter(r => r.available !== false))
const availabilitySummary = computed(() => availability.value
  ? `${displayRooms.value.filter(r => r.available).length} de ${displayRooms.value.length} habitaciones libres · ${availability.value.check_in} → ${availability.value.check_out}`
  : '')
const hotelName = ref('')
const hotels = ref([])
const activeHotelId = ref(null)
const propertyForm = reactive({ name: '', address: '', contact_email: '' })
const guests = ref([])
const guestSearch = ref('')
const guestDetail = ref(null)
const guestForm = reactive({ first_name: '', last_name: '', email: '', phone: '', document_type: 'cedula', document_number: '' })
const charges = ref([])
const msg = ref('')
const msgType = ref('ok')
const folioReservationId = ref(null)
const form = reactive({ guest_id: 0, guest_name: '', guest_email: '', room_id: 0, check_in: '', check_out: '', channel: 'direct' })
const charge = reactive({ description: '', amount: 0 })
const rateRules = ref([])
const rateForm = reactive({ room_id: 0, start_date: '', end_date: '', price: 0, notes: '' })
const quote = ref(null)
const quoteText = computed(() => {
  const q = quote.value
  if (!q) return 'Tarifa = noches × tarifa base'
  const dynamic = q.nights_detail.filter(n => !n.base).length
  return `${q.nights} noches · ${fmtCRC(q.total)} · ${dynamic} tarifa${dynamic === 1 ? '' : 's'} dinámica${dynamic === 1 ? '' : 's'}`
})
let timer = null

const me = ref({ username: '', role: 'admin' })
const users = ref([])
const userForm = reactive({ username: '', password: '', role: 'reception' })
const channelBookings = ref([])
const channelForm = reactive({ channel: 'airbnb', external_id: '', guest_name: '', room_id: 0, check_in: '', check_out: '' })
const hkTasks = ref([])
const hkFilter = ref('open')
const hkForm = reactive({ room_id: 0, type: 'checkout_clean', notes: '' })
const canClean = computed(() => ['admin', 'manager', 'reception', 'housekeeping'].includes(me.value.role))
const filteredHkTasks = computed(() => hkTasks.value.filter(t => hkFilter.value === 'all'
  ? true : hkFilter.value === 'open' ? t.status !== 'done' : t.status === hkFilter.value))
const hkStatusLabel = (s) => ({ pending: 'Pendiente', in_progress: 'En curso', done: 'Lista' }[s] || s)
const hkTypeLabel = (s) => ({ checkout_clean: 'Salida', stayover: 'Estancia', maintenance: 'Mantenimiento' }[s] || s)
const todayISO = new Date().toISOString().slice(0, 10)
const nextWeekISO = new Date(Date.now() + 7 * 86400000).toISOString().slice(0, 10)
const reportForm = reactive({ from: todayISO, to: nextWeekISO })
const report = ref(null)
const canAdmin = computed(() => me.value.role === 'admin')
const canRates = computed(() => ['admin', 'manager'].includes(me.value.role))
const canOps = computed(() => ['admin', 'manager', 'reception'].includes(me.value.role))
const canBilling = computed(() => ['admin', 'manager', 'accounting'].includes(me.value.role))
const token = ref(localStorage.getItem('hms_token') || '')
const needLogin = ref(false)
const activeView = ref('panel')
const viewLabels = {
  panel: 'Panel', reservas: 'Reservas', habitaciones: 'Habitaciones',
  huespedes: 'Huéspedes', folio: 'Folio · POSMaster', propiedades: 'Propiedades',
  tarifas: 'Tarifas dinámicas', canales: 'Canales OTA', limpieza: 'Limpieza',
  reportes: 'Reportes', usuarios: 'Usuarios y roles'
}
const pageTitle = computed(() => viewLabels[activeView.value] || 'Panel')
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
    // El login no debe pasar por apiFetch(): un 401 aquí son credenciales
    // inválidas, no una sesión expirada.
    const res = await fetch('/api/auth/login', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: String(loginForm.username || '').trim(),
        password: loginForm.password
      })
    })
    if (!res.ok) throw new Error('Credenciales inválidas')
    const data = await res.json()
    token.value = data.token
    localStorage.setItem('hms_token', data.token)
    loginForm.password = ''
    needLogin.value = false
    await loadMe()
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
const syncLabel = (s) => ({ local: 'Local', synced: 'Facturado', failed: 'Error sync', uncertain: 'Revisar', pending_sync: 'Enviando' }[s] || s)
const notify = (text, type = 'ok') => { msg.value = text; msgType.value = type; setTimeout(() => { msg.value = '' }, 5000) }

async function loadCharges() {
  if (!folioReservationId.value) { charges.value = []; return }
  try { charges.value = await apiFetch(`/api/reservations/${folioReservationId.value}/charges`).then(x => x.json()) }
  catch (e) { console.error(e) }
}

async function searchAvailability() {
  if (!search.check_in || !search.check_out) { notify('Elegí entrada y salida', 'err'); return }
  try {
    const res = await apiFetch(`/api/availability?check_in=${search.check_in}&check_out=${search.check_out}&hotel_id=${activeHotelId.value}`)
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    availability.value = await res.json()
    form.check_in = search.check_in
    form.check_out = search.check_out
    notify('Disponibilidad actualizada')
  } catch (e) { notify('No se pudo buscar: ' + e.message, 'err') }
}

async function load() {
  try {
    const hid = activeHotelId.value || ''
    const q = hid ? `?hotel_id=${hid}` : ''
    const [s, r, h, g, rates, identity, incoming, hk] = await Promise.all([
      apiFetch('/api/stats' + q).then(x => x.json()),
      apiFetch('/api/reservations' + q).then(x => x.json()),
      apiFetch('/api/hotels').then(x => x.json()),
      apiFetch('/api/guests' + q).then(x => x.json()),
      apiFetch('/api/rates' + q).then(x => x.json()),
      apiFetch('/api/auth/me').then(x => x.json()),
      apiFetch('/api/channels/bookings' + q).then(x => x.json()),
      apiFetch('/api/housekeeping/tasks' + q).then(x => x.json()),
    ])
    stats.value = s
    reservations.value = r
    hotels.value = h
    guests.value = g
    rateRules.value = rates
    me.value = identity
    channelBookings.value = incoming
    hkTasks.value = hk
    if (canAdmin.value) users.value = await apiFetch('/api/users').then(x => x.json())
    if (!activeHotelId.value && h.length) activeHotelId.value = h[0].ID
    const sel = h.find(x => x.ID === activeHotelId.value) || h[0]
    if (sel) {
      hotelName.value = sel.name
      rooms.value = await apiFetch(`/api/hotels/${sel.ID}/rooms`).then(x => x.json())
      if (!form.room_id && rooms.value.length) form.room_id = rooms.value[0].ID
      if (!channelForm.room_id) channelForm.room_id = rooms.value[0].ID
      if (!hkForm.room_id) hkForm.room_id = rooms.value[0].ID
      await loadQuote()
      await loadReport()
    }
    if (!folioReservationId.value && r.length) folioReservationId.value = r[0].ID
    await loadCharges()
  } catch (e) { console.error('API no disponible', e) }
}

function onGuestSelect() {
  const g = guests.value.find(x => x.ID === form.guest_id)
  if (g) {
    form.guest_name = [g.first_name, g.last_name].filter(Boolean).join(' ')
    form.guest_email = g.email || ''
  }
}

async function loadReport() {
  if (!activeHotelId.value || !reportForm.from || !reportForm.to) return
  try {
    const q = `hotel_id=${activeHotelId.value}&from=${reportForm.from}&to=${reportForm.to}`
    const res = await apiFetch(`/api/reports/summary?${q}`)
    if (res.ok) report.value = await res.json()
  } catch (e) { console.error(e) }
}

async function createHkTask() {
  try {
    const res = await apiFetch('/api/housekeeping/tasks', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ hotel_id: activeHotelId.value, ...hkForm })
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    notify('Tarea de limpieza creada')
    hkForm.notes = ''
    await load()
  } catch (e) { notify('No se pudo crear: ' + e.message, 'err') }
}

async function updateHkTask(t, status) {
  try {
    const res = await apiFetch(`/api/housekeeping/tasks/${t.ID}`, {
      method: 'PATCH', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status })
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    notify(status === 'done' ? 'Habitación disponible' : 'Tarea tomada')
    await load()
  } catch (e) { notify('No se pudo actualizar: ' + e.message, 'err') }
}

async function createChannelBooking() {
  try {
    const res = await apiFetch('/api/channels/bookings', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...channelForm, hotel_id: activeHotelId.value })
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    const data = await res.json()
    notify(data.duplicate ? 'Reserva ya importada (idempotente)' : `Reserva OTA #${data.reservation.ID} importada`)
    channelForm.external_id = ''; channelForm.guest_name = ''
    await load()
  } catch (e) { notify('No se pudo importar: ' + e.message, 'err') }
}

async function cancelChannelBooking(b) {
  try {
    const res = await apiFetch(`/api/channels/bookings/${b.ID}/cancel`, { method: 'POST' })
    if (!res.ok) throw new Error(res.status)
    notify('Reserva OTA cancelada')
    await load()
  } catch (e) { notify('No se pudo cancelar: ' + e.message, 'err') }
}

async function loadMe() {
  try { me.value = await apiFetch('/api/auth/me').then(x => x.json()) } catch (e) { console.error(e) }
}

async function createUser() {
  try {
    const res = await apiFetch('/api/users', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(userForm)
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    notify('Usuario creado')
    userForm.username = ''; userForm.password = ''
    users.value = await apiFetch('/api/users').then(x => x.json())
  } catch (e) { notify('No se pudo crear: ' + e.message, 'err') }
}

async function toggleUser(u) {
  try {
    const res = await apiFetch(`/api/users/${u.ID}`, {
      method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ active: !u.active })
    })
    if (!res.ok) throw new Error(res.status)
    users.value = await apiFetch('/api/users').then(x => x.json())
  } catch (e) { notify('No se pudo cambiar: ' + e.message, 'err') }
}

async function loadQuote() {
  quote.value = null
  if (!activeHotelId.value || !form.room_id || !form.check_in || !form.check_out) return
  if (form.check_out <= form.check_in) return
  try {
    const url = `/api/rates/quote?hotel_id=${activeHotelId.value}&room_id=${form.room_id}&check_in=${form.check_in}T14:00:00Z&check_out=${form.check_out}T12:00:00Z`
    const res = await apiFetch(url)
    if (res.ok) quote.value = await res.json()
  } catch (e) { console.error(e) }
}

async function createRate() {
  try {
    const res = await apiFetch('/api/rates', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ hotel_id: activeHotelId.value, ...rateForm })
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    notify('Tarifa agregada')
    rateForm.start_date = ''; rateForm.end_date = ''; rateForm.price = 0; rateForm.notes = ''
    await load()
  } catch (e) { notify('No se pudo agregar: ' + e.message, 'err') }
}

async function deleteRate(r) {
  try {
    const res = await apiFetch(`/api/rates/${r.ID}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(res.status)
    notify('Tarifa eliminada')
    await load()
  } catch (e) { notify('No se pudo eliminar: ' + e.message, 'err') }
}

async function createGuest() {
  try {
    const res = await apiFetch('/api/guests', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(Object.assign({ hotel_id: activeHotelId.value }, guestForm))
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    const created = await res.json()
    notify(`Huésped “${created.first_name}” registrado`)
    Object.keys(guestForm).forEach(k => { guestForm[k] = k === 'document_type' ? 'cedula' : '' })
    await load()
    form.guest_id = created.ID
    onGuestSelect()
  } catch (e) { notify('No se pudo registrar: ' + e.message, 'err') }
}

async function retryCharge(c) {
  try {
    const res = await apiFetch(`/api/reservations/${folioReservationId.value}/charges/${c.ID}/retry`, { method: 'POST' })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    const data = await res.json()
    notify(data.message)
    await loadCharges()
  } catch (e) { notify('No se pudo reintentar: ' + e.message, 'err') }
}

async function viewGuest(g) {
  try {
    const res = await apiFetch(`/api/guests/${g.ID}`)
    if (!res.ok) throw new Error(res.status)
    guestDetail.value = await res.json()
  } catch (e) { notify('No se pudo cargar el historial: ' + e.message, 'err') }
}

function onHotelChange() {
  guestDetail.value = null
  availability.value = null
  folioReservationId.value = null
  load()
}

async function createProperty() {
  try {
    const res = await apiFetch('/api/hotels', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(propertyForm)
    })
    if (!res.ok) { const e = await res.json(); throw new Error(e.error || res.status) }
    const created = await res.json()
    notify(`Propiedad “${created.name}” agregada`)
    propertyForm.name = ''; propertyForm.address = ''
    activeHotelId.value = created.ID
    availability.value = null
    await load()
  } catch (e) { notify('No se pudo agregar: ' + e.message, 'err') }
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
    hotel_id: activeHotelId.value, room_id: form.room_id, guest_id: form.guest_id, guest_name: form.guest_name, guest_email: form.guest_email,
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
.app { min-height: 100vh; display: grid; grid-template-columns: 232px minmax(0, 1fr); }
.workspace { display: flex; flex-direction: column; min-width: 0; }
.sidebar { background: #172033; border-right: 1px solid var(--border); display: flex; flex-direction: column; min-height: 100vh; }
.side-brand { display: flex; align-items: center; gap: 10px; padding: 18px 16px; border-bottom: 1px solid var(--border); }
.side-brand strong { display: block; font-size: 14px; }
.side-brand small { display: block; color: var(--muted); font-size: 10px; margin-top: 2px; }
.side-nav { padding: 12px 10px; flex: 1; }
.nav-group { color: var(--muted); font-size: 10px; letter-spacing: 1px; text-transform: uppercase; padding: 12px 10px 6px; }
.side-nav button { display: block; width: 100%; text-align: left; background: transparent; border: 0; border-left: 2px solid transparent; color: var(--muted); font: inherit; font-size: 13px; padding: 9px 10px; border-radius: 7px; cursor: pointer; margin-bottom: 2px; }
.side-nav button:hover { background: rgba(129,140,248,.08); color: var(--text); }
.side-nav button.active { background: rgba(99,102,241,.18); color: #c7d2fe; border-left-color: var(--primary); font-weight: 600; }
.side-foot { padding: 14px 16px; border-top: 1px solid var(--border); font-size: 10px; color: var(--muted); }
.page-title { font-size: 15px; font-weight: 700; white-space: nowrap; }
@media (max-width: 820px) { .app { grid-template-columns: 1fr; } .sidebar { display: none; } }

.topbar { display: flex; justify-content: space-between; align-items: center; padding: 16px 28px; border-bottom: 1px solid var(--border); background: var(--surface); }
.brand { display: flex; gap: 12px; align-items: center; }
.logo { font-size: 30px; }
h1 { font-size: 20px; letter-spacing: .3px; }
.sub { color: var(--muted); font-size: 12px; }
.hotel-select { max-width: 210px; }
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
.chip.received { background: rgba(167,139,250,.15); color: #c4b5fd; }
.chip.in_progress { background: rgba(96,165,250,.15); color: #60a5fa; }
.stats.mini { grid-template-columns: repeat(2, 1fr); margin: 12px 0; }
.report-table { width: 100%; border-collapse: collapse; font-size: 12px; margin-top: 8px; }
.report-table th, .report-table td { padding: 5px 6px; text-align: left; border-bottom: 1px solid rgba(148,163,184,.12); }
.chip.confirmed { background: rgba(96,165,250,.15); color: #60a5fa; }
.chip.checked_in { background: rgba(52,211,153,.15); color: #34d399; }
.chip.checked_out { background: rgba(148,163,184,.15); color: #94a3b8; }
.chip.canceled { background: rgba(248,113,113,.15); color: #f87171; }
.chip.sync { margin-left: auto; }
.chip.local { background: rgba(148,163,184,.15); color: #94a3b8; }
.chip.synced { background: rgba(52,211,153,.15); color: #34d399; }
.chip.uncertain { background: rgba(251,191,36,.15); color: #fbbf24; }
.chip.role { background: rgba(129,140,248,.18); color: #a5b4fc; text-transform: capitalize; }
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
.avail-search { display: flex; gap: 8px; margin-bottom: 10px; }
.avail-search input { flex: 1; min-width: 0; }
.room.blocked { opacity: .55; border-style: dashed; }
.room-blocked { font-size: 10px; color: #f87171; }
.dot { display: inline-block; width: 9px; height: 9px; border-radius: 50%; margin-right: 5px; }
.dot.available { background: #34d399; } .dot.occupied { background: #f87171; }
.dot.cleaning { background: #fbbf24; } .dot.maintenance { background: #64748b; }
.charge button { margin-left: auto; }
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
