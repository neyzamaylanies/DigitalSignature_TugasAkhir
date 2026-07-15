--
-- PostgreSQL database dump
--

\restrict NFf18DKq04tEPuhqyqX28H2fiiW28O44rP5ZHg6CccaZXUPHInQD9c5psXrAcUR

-- Dumped from database version 18.4
-- Dumped by pg_dump version 18.4

-- Started on 2026-07-15 10:12:19

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 222 (class 1259 OID 16614)
-- Name: dokumen; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.dokumen (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    judul character varying(255) NOT NULL,
    deskripsi text,
    pesan text,
    tipe character varying(100),
    jenis character varying(50) NOT NULL,
    file_path character varying(255) NOT NULL,
    final_file_path character varying(255),
    status character varying(50) DEFAULT 'draft'::character varying,
    created_at timestamp with time zone
);


ALTER TABLE public.dokumen OWNER TO postgres;

--
-- TOC entry 221 (class 1259 OID 16613)
-- Name: dokumen_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.dokumen_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.dokumen_id_seq OWNER TO postgres;

--
-- TOC entry 5066 (class 0 OID 0)
-- Dependencies: 221
-- Name: dokumen_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.dokumen_id_seq OWNED BY public.dokumen.id;


--
-- TOC entry 232 (class 1259 OID 16684)
-- Name: log_aktivitas; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.log_aktivitas (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    dokumen_id bigint,
    aksi character varying(100) NOT NULL,
    keterangan text,
    created_at timestamp with time zone
);


ALTER TABLE public.log_aktivitas OWNER TO postgres;

--
-- TOC entry 231 (class 1259 OID 16683)
-- Name: log_aktivitas_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.log_aktivitas_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.log_aktivitas_id_seq OWNER TO postgres;

--
-- TOC entry 5067 (class 0 OID 0)
-- Dependencies: 231
-- Name: log_aktivitas_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.log_aktivitas_id_seq OWNED BY public.log_aktivitas.id;


--
-- TOC entry 224 (class 1259 OID 16629)
-- Name: permintaan_ttd; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.permintaan_ttd (
    id bigint NOT NULL,
    dokumen_id bigint NOT NULL,
    user_id bigint NOT NULL,
    urutan bigint NOT NULL,
    page_number bigint NOT NULL,
    width numeric,
    height numeric,
    status character varying(50) DEFAULT 'menunggu'::character varying,
    alasan_tolak text,
    created_at timestamp with time zone
);


ALTER TABLE public.permintaan_ttd OWNER TO postgres;

--
-- TOC entry 223 (class 1259 OID 16628)
-- Name: permintaan_ttd_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.permintaan_ttd_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.permintaan_ttd_id_seq OWNER TO postgres;

--
-- TOC entry 5068 (class 0 OID 0)
-- Dependencies: 223
-- Name: permintaan_ttd_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.permintaan_ttd_id_seq OWNED BY public.permintaan_ttd.id;


--
-- TOC entry 226 (class 1259 OID 16644)
-- Name: sertifikat; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.sertifikat (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    serial_number character varying(100) NOT NULL,
    public_key text NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying,
    valid_from timestamp with time zone,
    valid_until timestamp with time zone,
    created_at timestamp with time zone
);


ALTER TABLE public.sertifikat OWNER TO postgres;

--
-- TOC entry 225 (class 1259 OID 16643)
-- Name: sertifikat_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.sertifikat_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sertifikat_id_seq OWNER TO postgres;

--
-- TOC entry 5069 (class 0 OID 0)
-- Dependencies: 225
-- Name: sertifikat_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.sertifikat_id_seq OWNED BY public.sertifikat.id;


--
-- TOC entry 228 (class 1259 OID 16660)
-- Name: tanda_tangan; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tanda_tangan (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    tipe character varying(50) NOT NULL,
    file_path character varying(255) NOT NULL,
    created_at timestamp with time zone
);


ALTER TABLE public.tanda_tangan OWNER TO postgres;

--
-- TOC entry 227 (class 1259 OID 16659)
-- Name: tanda_tangan_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.tanda_tangan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.tanda_tangan_id_seq OWNER TO postgres;

--
-- TOC entry 5070 (class 0 OID 0)
-- Dependencies: 227
-- Name: tanda_tangan_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.tanda_tangan_id_seq OWNED BY public.tanda_tangan.id;


--
-- TOC entry 230 (class 1259 OID 16671)
-- Name: transaksi_sertifikat; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaksi_sertifikat (
    id bigint NOT NULL,
    permintaan_ttd_id bigint,
    dokumen_id bigint NOT NULL,
    user_id bigint NOT NULL,
    sertifikat_id bigint,
    aksi character varying(50) NOT NULL,
    alasan_tolak text,
    file_result_path character varying(255),
    created_at timestamp with time zone
);


ALTER TABLE public.transaksi_sertifikat OWNER TO postgres;

--
-- TOC entry 229 (class 1259 OID 16670)
-- Name: transaksi_sertifikat_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.transaksi_sertifikat_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.transaksi_sertifikat_id_seq OWNER TO postgres;

--
-- TOC entry 5071 (class 0 OID 0)
-- Dependencies: 229
-- Name: transaksi_sertifikat_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.transaksi_sertifikat_id_seq OWNED BY public.transaksi_sertifikat.id;


--
-- TOC entry 220 (class 1259 OID 16598)
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    email character varying(100) NOT NULL,
    password character varying(255) NOT NULL,
    role character varying(50) NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.users OWNER TO postgres;

--
-- TOC entry 219 (class 1259 OID 16597)
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO postgres;

--
-- TOC entry 5072 (class 0 OID 0)
-- Dependencies: 219
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- TOC entry 4887 (class 2604 OID 16617)
-- Name: dokumen id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.dokumen ALTER COLUMN id SET DEFAULT nextval('public.dokumen_id_seq'::regclass);


--
-- TOC entry 4895 (class 2604 OID 16687)
-- Name: log_aktivitas id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.log_aktivitas ALTER COLUMN id SET DEFAULT nextval('public.log_aktivitas_id_seq'::regclass);


--
-- TOC entry 4889 (class 2604 OID 16632)
-- Name: permintaan_ttd id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permintaan_ttd ALTER COLUMN id SET DEFAULT nextval('public.permintaan_ttd_id_seq'::regclass);


--
-- TOC entry 4891 (class 2604 OID 16647)
-- Name: sertifikat id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sertifikat ALTER COLUMN id SET DEFAULT nextval('public.sertifikat_id_seq'::regclass);


--
-- TOC entry 4893 (class 2604 OID 16663)
-- Name: tanda_tangan id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tanda_tangan ALTER COLUMN id SET DEFAULT nextval('public.tanda_tangan_id_seq'::regclass);


--
-- TOC entry 4894 (class 2604 OID 16674)
-- Name: transaksi_sertifikat id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat ALTER COLUMN id SET DEFAULT nextval('public.transaksi_sertifikat_id_seq'::regclass);


--
-- TOC entry 4886 (class 2604 OID 16601)
-- Name: users id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- TOC entry 4901 (class 2606 OID 16627)
-- Name: dokumen dokumen_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.dokumen
    ADD CONSTRAINT dokumen_pkey PRIMARY KEY (id);


--
-- TOC entry 4913 (class 2606 OID 16694)
-- Name: log_aktivitas log_aktivitas_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.log_aktivitas
    ADD CONSTRAINT log_aktivitas_pkey PRIMARY KEY (id);


--
-- TOC entry 4903 (class 2606 OID 16642)
-- Name: permintaan_ttd permintaan_ttd_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permintaan_ttd
    ADD CONSTRAINT permintaan_ttd_pkey PRIMARY KEY (id);


--
-- TOC entry 4905 (class 2606 OID 16656)
-- Name: sertifikat sertifikat_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sertifikat
    ADD CONSTRAINT sertifikat_pkey PRIMARY KEY (id);


--
-- TOC entry 4909 (class 2606 OID 16669)
-- Name: tanda_tangan tanda_tangan_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tanda_tangan
    ADD CONSTRAINT tanda_tangan_pkey PRIMARY KEY (id);


--
-- TOC entry 4911 (class 2606 OID 16682)
-- Name: transaksi_sertifikat transaksi_sertifikat_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat
    ADD CONSTRAINT transaksi_sertifikat_pkey PRIMARY KEY (id);


--
-- TOC entry 4907 (class 2606 OID 16658)
-- Name: sertifikat uni_sertifikat_serial_number; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sertifikat
    ADD CONSTRAINT uni_sertifikat_serial_number UNIQUE (serial_number);


--
-- TOC entry 4897 (class 2606 OID 16612)
-- Name: users uni_users_email; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT uni_users_email UNIQUE (email);


--
-- TOC entry 4899 (class 2606 OID 16610)
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


-- Completed on 2026-07-15 10:12:20

--
-- PostgreSQL database dump complete
--

\unrestrict NFf18DKq04tEPuhqyqX28H2fiiW28O44rP5ZHg6CccaZXUPHInQD9c5psXrAcUR

