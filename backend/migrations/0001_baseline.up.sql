-- HMSMaster baseline schema. Idempotent: safe on an existing database.
CREATE TABLE IF NOT EXISTS public.channel_bookings (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hotel_id bigint NOT NULL,
    channel character varying(20) NOT NULL,
    external_id character varying(120) NOT NULL,
    reservation_id bigint,
    guest_name text,
    check_in timestamp with time zone,
    check_out timestamp with time zone,
    status character varying(20) DEFAULT 'received'::character varying,
    payload text
);

CREATE TABLE IF NOT EXISTS public.folio_charges (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    reservation_id bigint NOT NULL,
    description text NOT NULL,
    amount numeric NOT NULL,
    source text DEFAULT 'hms'::text,
    pos_ref text,
    sync_status text DEFAULT 'local'::text,
    idempotency_key character varying(64),
    sync_attempts bigint,
    sync_error character varying(500)
);

CREATE TABLE IF NOT EXISTS public.guests (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hotel_id bigint NOT NULL,
    first_name text NOT NULL,
    last_name text,
    email text,
    phone text,
    document_type text,
    document_number text,
    notes text
);

CREATE TABLE IF NOT EXISTS public.hotels (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL,
    address text NOT NULL,
    contact_email text,
    phone text,
    active boolean DEFAULT true
);

CREATE TABLE IF NOT EXISTS public.housekeeping_tasks (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hotel_id bigint NOT NULL,
    room_id bigint NOT NULL,
    room_number text,
    reservation_id bigint,
    type character varying(30) DEFAULT 'checkout_clean'::character varying,
    status character varying(20) DEFAULT 'pending'::character varying,
    assigned_to character varying(80),
    notes character varying(500),
    completed_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.rate_rules (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hotel_id bigint NOT NULL,
    room_id bigint,
    start_date character varying(10) NOT NULL,
    end_date character varying(10) NOT NULL,
    price numeric NOT NULL,
    notes character varying(255)
);

CREATE TABLE IF NOT EXISTS public.reservations (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hotel_id bigint NOT NULL,
    room_id bigint,
    guest_name text NOT NULL,
    guest_email text,
    check_in timestamp with time zone NOT NULL,
    check_out timestamp with time zone NOT NULL,
    room_number text,
    total_amount numeric NOT NULL,
    status text DEFAULT 'pending'::text,
    payment_status text DEFAULT 'unpaid'::text,
    channel text DEFAULT 'direct'::text,
    guest_id bigint,
    rate_detail text
);

CREATE TABLE IF NOT EXISTS public.rooms (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hotel_id bigint NOT NULL,
    number text NOT NULL,
    type text DEFAULT 'standard'::text,
    floor bigint,
    base_rate numeric NOT NULL,
    status text DEFAULT 'available'::text
);

CREATE TABLE IF NOT EXISTS public.users (
    id bigserial PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    username character varying(80) NOT NULL,
    password_hash character varying(100) NOT NULL,
    role character varying(20) DEFAULT 'reception'::character varying NOT NULL,
    hotel_id bigint,
    active boolean DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_channel_bookings_channel ON public.channel_bookings USING btree (channel);
CREATE INDEX IF NOT EXISTS idx_channel_bookings_deleted_at ON public.channel_bookings USING btree (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_bookings_external ON public.channel_bookings USING btree (channel, external_id);
CREATE INDEX IF NOT EXISTS idx_channel_bookings_hotel_id ON public.channel_bookings USING btree (hotel_id);
CREATE INDEX IF NOT EXISTS idx_channel_bookings_reservation_id ON public.channel_bookings USING btree (reservation_id);
CREATE INDEX IF NOT EXISTS idx_folio_charges_deleted_at ON public.folio_charges USING btree (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_folio_charges_idempotency_key ON public.folio_charges USING btree (idempotency_key);
CREATE INDEX IF NOT EXISTS idx_folio_charges_reservation_id ON public.folio_charges USING btree (reservation_id);
CREATE INDEX IF NOT EXISTS idx_guests_deleted_at ON public.guests USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_guests_document_number ON public.guests USING btree (document_number);
CREATE INDEX IF NOT EXISTS idx_guests_email ON public.guests USING btree (email);
CREATE INDEX IF NOT EXISTS idx_guests_hotel_id ON public.guests USING btree (hotel_id);
CREATE INDEX IF NOT EXISTS idx_hotels_deleted_at ON public.hotels USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_housekeeping_tasks_deleted_at ON public.housekeeping_tasks USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_housekeeping_tasks_hotel_id ON public.housekeeping_tasks USING btree (hotel_id);
CREATE INDEX IF NOT EXISTS idx_housekeeping_tasks_reservation_id ON public.housekeeping_tasks USING btree (reservation_id);
CREATE INDEX IF NOT EXISTS idx_housekeeping_tasks_room_id ON public.housekeeping_tasks USING btree (room_id);
CREATE INDEX IF NOT EXISTS idx_rate_rules_deleted_at ON public.rate_rules USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_rate_rules_end_date ON public.rate_rules USING btree (end_date);
CREATE INDEX IF NOT EXISTS idx_rate_rules_hotel_id ON public.rate_rules USING btree (hotel_id);
CREATE INDEX IF NOT EXISTS idx_rate_rules_room_id ON public.rate_rules USING btree (room_id);
CREATE INDEX IF NOT EXISTS idx_rate_rules_start_date ON public.rate_rules USING btree (start_date);
CREATE INDEX IF NOT EXISTS idx_reservations_deleted_at ON public.reservations USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_reservations_guest_id ON public.reservations USING btree (guest_id);
CREATE INDEX IF NOT EXISTS idx_reservations_hotel_id ON public.reservations USING btree (hotel_id);
CREATE INDEX IF NOT EXISTS idx_reservations_room_id ON public.reservations USING btree (room_id);
CREATE INDEX IF NOT EXISTS idx_reservations_room_number ON public.reservations USING btree (room_number);
CREATE INDEX IF NOT EXISTS idx_rooms_deleted_at ON public.rooms USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_rooms_hotel_id ON public.rooms USING btree (hotel_id);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON public.users USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_hotel_id ON public.users USING btree (hotel_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON public.users USING btree (username);

CREATE UNIQUE INDEX IF NOT EXISTS idx_folio_charges_idempotency_key_partial
  ON public.folio_charges (idempotency_key) WHERE idempotency_key <> '';
