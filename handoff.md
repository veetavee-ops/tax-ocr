# Tax OCR System — Handoff Document
> อัปเดต: 2026-06-23 | Session 16

---

## 1. Credentials & Local Dev

### Credentials (ไม่ขึ้น git — เก็บใน Google Drive)
> https://drive.google.com/drive/folders/17BDc1uAofvv5irAeaf4pgVRxMV7AqP2l

ดาวน์โหลดไฟล์จาก Drive แล้ววางที่:
- `infrastructure/.env` — credentials สำหรับ Docker services
- `backend/.env` — credentials สำหรับ backend app (API keys, JWT secret ฯลฯ)

### Service Ports
| Service | Port | URL |
|---------|------|-----|
| Admin UI | 3000 | http://localhost:3000 |
| LIFF (User UI) | 5174 | http://localhost:5174 |
| Backend API | 8080 | http://localhost:8080/api/v1 |
| PostgreSQL | 5433 | localhost:5433 |
| Redis | 6380 | localhost:6380 |
| MinIO Console | 9001 | http://localhost:9001 |
| MinIO API | 9000 | http://localhost:9000 |

---

## 2. วิธีรัน Local Dev

### ขั้นตอน (เรียงตาม order)

**Step 1 — เปิด Docker Desktop** (ต้องเห็น icon สีเขียวใน system tray)

**Step 2 — Start infrastructure**
```powershell
Set-Location e:\tax-ocr\infrastructure
docker compose up -d
```

**Step 3 — Start backend** (auto-migrate on startup)
```powershell
Set-Location e:\tax-ocr\backend
go run ./cmd/
# port 8080, auto-migrate ทุก migration ที่ยังไม่ได้ run
```
> ⚠️ ใช้ `./cmd/` ไม่ใช่ `./cmd/...` (มี 2 packages: main + migrate)

**Step 4 — Start Admin UI**
```powershell
Set-Location e:\tax-ocr\frontend\admin
npm run dev
# port 3000
```

**Step 5 — Start LIFF (optional)**
```powershell
Set-Location e:\tax-ocr\frontend\liff
npm run dev
# port 5174
```

### Migration (manual)
```powershell
# ครั้งแรกบน DB เก่าที่ไม่มี schema_migrations table
$env:MIGRATIONS_DIR = "e:\tax-ocr\database\migrations"
Set-Location e:\tax-ocr\backend
go run ./cmd/migrate/ -stamp

# apply migrations ใหม่
go run ./cmd/migrate/
```

### DB Shell
```powershell
docker exec -it tax-ocr-postgres psql -U tax_ocr -d tax_ocr
```

---

## 3. Git Status
```
Branch: master
Remote: https://github.com/veetavee-ops/tax-ocr.git
Latest: 55f748f docs: update CLAUDE.md + handoff.md session 14-15
        cb093cc Session 14-15: tenant trash/suspend, modal UX, PDF OCR, suspend enforcement
        3ba3d7b handoff: add e-Tax XML support to next tasks
        8d5d303 Update CLAUDE.md session 13 + handoff.md
        796e8eb Fix: OCR company accept images only (JPG/PNG), not PDF
```
Clone สำรองไว้ที่ `d:\tax-ocr` (local clone)

---

## 4. Architecture Overview

### Tech Stack
| Layer | Tech |
|-------|------|
| User UI | LINE LIFF + React + Vite |
| Admin UI | React + Vite + Tailwind |
| Backend | Go (Golang) |
| Queue | Asynq + Redis |
| Database | PostgreSQL |
| Object Storage | MinIO |
| OCR Engine 1 | Google Cloud Vision (text reading) |
| OCR Engine 2 | GPT-4o-mini (structure extraction) |

### OCR Flow (final architecture — ห้ามเปลี่ยน)
```
ลูกค้าส่งไฟล์ → MinIO → Asynq Queue
→ Worker:
  1. Vision API → raw Thai text + classifyFromText() → doc_type, vat_inclusive
  2. GPT-4o-mini → รับ text + VISION HINT → extract ทุก field (sole authority)
  3. crossVerify → เปรียบ vendor_tax_id, totals
  4. validateBuyer → ตรวจ buyer vs tenant/branch (tax_invoice เท่านั้น)
  5. duplicate check → vendor_tax_id + invoice_doc_no
  6. vendor upsert → link vendor_id
  7. classify items → rule → AI → HITL
→ LINE push แจ้งลูกค้า
```

### Buyer Validation Rules (ม.82/5)
| Field | Rule | ผิด → |
|-------|------|--------|
| buyer_tax_id | exact match กับ tenant.tax_id | status = `invalid` |
| buyer_branch_code | normalized exact match กับ branch.code | status = `invalid` |
| buyer_name | Levenshtein similarity ≥ 85% | status = `invalid` |
| invoice_date | ≤ 90 วันจากวันนี้ | invalid_reason = `late_invoice_vat_unclaimed` (status คงเดิม) |

Branch code normalization: `"สำนักงานใหญ่"`, `"HQ"`, `"0"` → `"00000"`

---

## 5. Database Schema (current — session 12)

### tenants
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| name | VARCHAR(255) | ชื่อบริษัท |
| tax_id | VARCHAR(13) | เลขผู้เสียภาษี UNIQUE |
| address | TEXT | ที่อยู่จดทะเบียน (สำหรับ header รายงานภาษี) |
| business_type | VARCHAR(20) | trading / service / construction |
| status | VARCHAR(20) | active / inactive |

### branches
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| name, code | VARCHAR | UNIQUE(tenant_id, code) |
| address | TEXT | ที่อยู่สาขา |
| phone | VARCHAR(20) | เบอร์โทรสาขา |
| status | VARCHAR(20) | active / inactive |

### invoices (key fields)
| Column | Type | Notes |
|--------|------|-------|
| status | VARCHAR(20) | **pending / verified / conflict / invalid** |
| invalid_reason | TEXT | buyer_tax_id_mismatch / buyer_branch_code_mismatch / buyer_name_mismatch / late_invoice_vat_unclaimed |
| doc_type | VARCHAR(50) | tax_invoice / receipt / invoice_billing / delivery_order |
| vat_inclusive | BOOLEAN | ราคารวม VAT แล้ว? |
| invoice_doc_no | TEXT | เลขที่ใบกำกับ |
| invoice_date | TEXT | วันที่ในเอกสาร |
| invoice_year/month/day | INT | parsed CE year |
| accounting_year/month | INT | รอบบัญชี ภพ.30 |
| duplicate_of | UUID | FK → invoices nullable |
| vendor_id | UUID | FK → vendors nullable |

### vendors
| Column | Notes |
|--------|-------|
| tax_id | UNIQUE |
| verified | bool — ยืนยันโดย admin แล้ว? |

---

## 6. Migrations Applied (34 total)
```
001–021  Core schema (tenants, branches, users, invoices, items, etc.)
022      add vendor_name to invoices
023      add invoice_no (serial)
024      add vat_math_ok
025      add invoice_doc fields (doc_no, date, buyer/vendor info)
026      add invoice verified (verified_by, verified_at)
027      expand invoice fields (vat_inclusive, vat_rate, etc.)
028      expand invoice_item fields (product_code, unit, discount)
029      add invoice_date_parts + duplicate_of
030      add accounting_period (accounting_year, accounting_month)
031      create vendors table
032      add invalid_reason + business_type  ← session 12
033      add address (tenants/branches) + phone (branches)  ← session 12
034      tenant soft-delete + suspend (deleted_at, suspended_at, suspension_reason)  ← session 14
```

---

## 7. Key Files

### Backend
```
backend/
├── cmd/main.go                    # entry point, wire everything
├── cmd/migrate/main.go            # migration CLI
├── internal/api/
│   ├── router.go                  # all routes + tenant/branch handlers
│   ├── invoice_handler.go         # invoice CRUD
│   ├── document_handler.go        # upload / link
│   ├── vendor_handler.go          # vendor CRUD + verify
│   └── auth_handler.go            # login / register / refresh
├── internal/db/
│   ├── store.go                   # structs + errors
│   ├── tenant_store.go            # tenant + branch queries
│   ├── invoice_store.go           # invoice + item queries
│   └── vendor_store.go            # vendor queries
├── internal/queue/
│   ├── worker.go                  # main processing: OCR → validate → classify
│   └── task.go                    # ProcessInvoicePayload
├── internal/ocr/
│   ├── service.go                 # orchestrates dual-engine
│   ├── gpt.go                     # GPT-4o-mini extraction
│   ├── vision.go                  # Google Cloud Vision + classifyFromText
│   ├── crossverify.go             # cross-verify logic
│   └── company.go                 # OCR company extract (JPG/PNG/PDF → CompanyData)
└── internal/classify/
    └── service.go                 # rule → AI → HITL classification
```

### Frontend Admin
```
frontend/admin/src/pages/
├── Invoices.jsx         # list + filter + upload modal
├── InvoiceDetail.jsx    # detail + edit + verification wizard
├── Tenants.jsx          # CRUD tenant + business_type + address
├── Branches.jsx         # CRUD branch + address + phone
├── Vendors.jsx          # vendor list + verify
└── ...
```

---

## 8. Invoice Status Flow
```
pending   → OCR กำลังทำงาน (auto-refresh 3s)
verified  → OCR ผ่าน + buyer valid (อาจมี invalid_reason=late_invoice เป็น warning)
conflict  → OCR cross-verify ไม่ตรง → ต้องแก้มือ
invalid   → buyer info ไม่ตรงกับ tenant/branch → ภาษีซื้อต้องห้าม ม.82/5
```

---

## 9. สิ่งที่ยังต้องทำ (Next Tasks)

### ทำได้ทันที
- [ ] ทดสอบ buyer validation — อัปโหลดใบที่ buyer_tax_id ผิด → ควรเห็น `invalid` ใน UI
- [ ] ทดสอบ OCR company extract — JPG/PNG/PDF หนังสือรับรอง → auto-fill P-01-M Create
- [ ] GPT prompt invoice: เพิ่ม `invoice_billing`/`delivery_order` ใน classification
- [ ] **Google Drive overwrite script** — Service Account + Python script (`~/.claude/scripts/gdrive-update.py`) เพื่อให้ `/mem` เขียนทับไฟล์ใน Drive ได้แทนการ create ใหม่ทุกครั้ง → ใช้ได้ทุกโปรเจกต์

### Phase ถัดไป
- [ ] **e-Tax Invoice XML support** — ผู้ขายส่ง XML มาตรฐาน RD แทนรูป → parse XML → บันทึก invoice (ข้าม OCR), ต้องมีไฟล์ตัวอย่างจาก vendor ก่อน
  - Signature validation ทำทีหลัง (ต้องใช้ cert RD)
  - Export ภพ.30 XML → Phase ถัดไป
- [ ] รายงานภาษีซื้อ (ม.87/1) — export PDF/Excel พร้อม header ที่อยู่
- [ ] PDF OCR support (invoice — scanned PDF)
- [x] PDF company extract — digital PDF จาก DBD → Go PDF library extract text → GPT ✅ session 15
- [ ] Password reset flow
- [ ] OneDrive API integration

### Production (ยังไม่ถึงเวลา — อย่าทำ)
- Dockerfile x3 (backend, admin, liff)
- nginx + SSL
- LINE OA production config
- Target: Hetzner CX22 (~€4/เดือน)

---

## 10. Rules & Gotchas

**DB:** อย่า `CASE WHEN $n != 0 THEN $n` กับ float columns — PostgreSQL infer เป็น bigint → ตัดทศนิยม ใช้ `SET col = $n` ตรงๆ เสมอ

**OCR:** Vision = อ่านข้อความ + classify เท่านั้น, GPT = extract values เท่านั้น อย่าให้ Vision extract ตัวเลข

**Backend run:** `go run ./cmd/` ไม่ใช่ `./cmd/...`

**Migration:** ต้องตั้ง `$env:MIGRATIONS_DIR` ก่อน run migrate CLI เพราะ relative path จาก `backend/` ไปไม่ถึง

**address fields:** ไม่ใช้ใน OCR buyer validation — เก็บไว้สำหรับ header รายงานภาษีซื้อเท่านั้น

**Tenant suspend:** `checkTenantStatus` middleware ตรวจ DB ทุก request — suspended tenant โดน 403 ทันที login/refresh ก็โดนด้วย ปุ่ม "ปิดบริการ" แสดงเฉพาะ `status === 'active'`

**Popup pattern:** ทุก confirm action ใช้ `ConfirmDialog` + `confirmXxx` state + `-my-3 gap-0 py-3` สำหรับ full-row click area
- Skill: `/popup` (ไฟล์: `~/.claude/commands/popup.md`) — เดิมชื่อ `/ui-modal`
- Components: `useDblClickProtect` hook + `ConfirmDialog` component (วางต้นไฟล์ page เดียวกัน)
- ปุ่มใน table: wrapper `-my-3 gap-0`, ปุ่ม `py-3 px-3` = full-row click area

**OCR API Keys:** เก็บใน DB (`ocr_config` table) ไม่ใช่ `.env` — ถ้า DB volume ถูก wipe ต้องกรอกใหม่ใน Admin UI → Settings → OCR Config

**Credentials:** ห้ามถามผู้ใช้ — อ่านจาก Google Drive เสมอ (`/creds` skill)
- Drive root: `https://drive.google.com/drive/folders/17BDc1uAofvv5irAeaf4pgVRxMV7AqP2l`
- Project folder: `tax-ocr/` → ไฟล์ `.env` และ `handoff.md`

**Custom Skills:** `~/.claude/commands/` — `/creds`, `/mem`, `/popup`, `/helpskill`
