CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS users (id uuid PRIMARY KEY,mobile varchar(20) UNIQUE NOT NULL,name varchar(120),location varchar(255),plan varchar(20) NOT NULL DEFAULT 'BASIC',status varchar(20) NOT NULL DEFAULT 'ACTIVE',created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS partners (id uuid PRIMARY KEY,name varchar(200) NOT NULL,type varchar(50) NOT NULL,city varchar(120),phone varchar(30),address text,benefits text,status varchar(20) NOT NULL DEFAULT 'PENDING',created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS idx_partners_type_city ON partners(type,city);
CREATE TABLE IF NOT EXISTS medical_help_requests (id uuid PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id),details text NOT NULL,status varchar(30) NOT NULL DEFAULT 'SUBMITTED',created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS idx_medical_help_user_created ON medical_help_requests(user_id,created_at DESC);
CREATE TABLE IF NOT EXISTS offers_updates (id uuid PRIMARY KEY,title varchar(200) NOT NULL,description text NOT NULL,location varchar(255),starts_at timestamptz,ends_at timestamptz,status varchar(30) NOT NULL DEFAULT 'DRAFT',created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS idx_offers_status_created ON offers_updates(status,created_at DESC);
