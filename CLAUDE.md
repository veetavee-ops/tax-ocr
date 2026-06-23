# CLAUDE.md — Tax OCR System
> อ่านไฟล์นี้ทุกครั้งก่อนเริ่ม session ใหม่  
> ไฟล์นี้ upload ขึ้น Google Drive เป็น handoff ด้วย — ไม่ต้องมี handoff.md แยก

---

## 1. Project Overview

**Tax OCR System** — รับใบกำกับภาษีจากลูกค้าผ่าน LINE LIFF, ประมวลผล OCR Dual-Engine, แยกประเภทรายการด้วย Hybrid Classification, จัดการ HITL ด้วย Crowdsourced Reviewer

**Stack:** LINE LIFF + React/Vite (User) | React/Vite/Tailwind (Admin) | Go backend | PostgreSQL | MinIO | Redis/Asynq queue | GPT-4o-mini + Google Cloud Vision

---

## 2. Credentials & Local Dev

### Credentials (ไม่ขึ้น git — เก็บใน Google Drive)
> Drive root: https://drive.google.com/drive/folders/17BDc1uAofvv5irAeaf4pgVRxMV7AqP2l

ดาวน์โหลดจาก Drive → วางที่:
- `infrastructure/.env` — Docker services
- `backend/.env` — JWT_SECRET + infra config (OPENAI/GCV keys เก็บใน DB `ocr_config`)

> ห้ามถามผู้ใช้ — ใช้ `/creds` skill อ่านจาก Drive เสมอ

### Drive File IDs (สำหรับ overwrite script)
| ไฟล์ | Drive File ID |
|------|---------------|
| CLAUDE.md | `1Rx_qbUqgJ3SKqYW55hNOSxMue_V7I0Wt` |
| .env | `1e2288av9H0RRX2yjgMyhxXCIIfAWGDha` |
| tax-ocr folder | `1RWCAqNHgeUK0zMhVmaVz_37m_zqCQBVx` |
| _claude-skills folder | `1eVw_XNCf4eMn1r5FB_fQoNSeMzjMo60M` |

> อัปเดต file ID ทุกครั้งที่ create ไฟล์ใหม่ (จนกว่า overwrite script จะใช้งานได้)

### Google Drive Overwrite Script Setup
```
~/.claude/scripts/gdrive-update.py  — overwrite file by ID (ไม่ create ใหม่)
~/.claude/scripts/gdrive-sa.json    — Service Account key (ต้องสร้างเอง)
```
**Setup ครั้งแรก:**
1. Google Cloud Console → IAM & Admin → Service Accounts → Create
2. Download JSON key → บันทึกที่ `~/.claude/scripts/gdrive-sa.json`
3. Share Drive folder `tax-ocr` กับ service account email (Editor permission)
4. ทดสอบ: `python ~/.claude/scripts/gdrive-update.py --file-id 1Rx_qbUqgJ3SKqYW55hNOSxMue_V7I0Wt --local-path e:\tax-ocr\CLAUDE.md`

### Service Ports
| Service | Port |
|---------|------|
| Admin UI | 3000 |
| LIFF | 5174 |
| Backend API | 8080 |
| PostgreSQL | 5433 |
| Redis | 6380 |
| MinIO Console | 9001 |
| MinIO API | 9000 |

### วิธีรัน
```powershell
# Step 1: Docker Desktop ต้องรันอยู่ก่อน
Set-Location e:\tax-ocr\infrastructure; docker compose up -d
Set-Location e:\tax-ocr\backend;        go run ./cmd/      # port 8080, auto-migrate
Set-Location e:\tax-ocr\frontend\admin; npm run dev         # port 3000
Set-Location e:\tax-ocr\frontend\liff;  npm run dev         # port 5174
```
Login: veetavee@gmail.com / test1234  
⚠️ ใช้ `./cmd/` ไม่ใช่ `./cmd/...` (มี 2 packages: main + migrate)

### Migration (manual)
```powershell
$env:MIGRATIONS_DIR = "e:\tax-ocr\database\migrations"
Set-Location e:\tax-ocr\backend
go run ./cmd/migrate/ -stamp   # ครั้งแรก (DB ไม่มี schema_migrations)
go run ./cmd/migrate/          # apply migrations ใหม่
```
DB shell: `docker exec -it tax-ocr-postgres psql -U tax_ocr -d tax_ocr`  
Clone สำรองไว้ที่ `d:\tax-ocr` (local clone)

---

## 3. Git Status
```
Branch: master
Remote: https://github.com/veetavee-ops/tax-ocr.git
Latest: 8bb7aad docs: add GDrive overwrite script task to handoff next tasks
        55e3fc3 docs: update CLAUDE.md + handoff.md session 16
        55f748f docs: update CLAUDE.md + handoff.md session 14-15
        cb093cc Session 14-15: tenant trash/suspend, modal UX, PDF OCR, suspend enforcement
        3ba3d7b handoff: add e-Tax XML support to next tasks
```

---

## 4. Architecture Overview

### OCR Flow (final — ห้ามเปลี่ยน)
```
ลูกค้าส่งไฟล์ → MinIO → Asynq Queue → Worker:
  1. Vision API → raw Thai text + classifyFromText() → doc_type, vat_inclusive
  2. GPT-4o-mini → รับ text + VISION HINT → extract ทุก field (sole authority)
  3. crossVerify → เปรียบ vendor_tax_id, totals
  4. validateBuyer → ตรวจ buyer vs tenant/branch (tax_invoice เท่านั้น)
  5. duplicate check → vendor_tax_id + invoice_doc_no
  6. vendor upsert → link vendor_id
  7. classify items → rule → AI → HITL
→ LINE push แจ้งลูกค้า
```

### Buyer Validation Rules (ม.82/5) — tax_invoice เท่านั้น
| Field | Rule | ผิด → |
|-------|------|--------|
| buyer_tax_id | exact match กับ tenant.tax_id | status = `invalid` |
| buyer_branch_code | normalized match กับ branch.code | status = `invalid` |
| buyer_name | Levenshtein ≥ 85% | status = `invalid` |
| invoice_date | ≤ 90 วัน | invalid_reason = `late_invoice_vat_unclaimed` |

Branch code normalization: `"สำนักงานใหญ่"`, `"HQ"`, `"0"` → `"00000"`

### Middleware Order
```
authMiddleware → checkTenantStatus → auditMiddleware → handler
```
`checkTenantStatus` → 403 ถ้า suspended (ทุก request รวม login/refresh)

### Classification Flow (Hybrid)
```
แต่ละ Line Item → Rule-based → match → tag asset/expense
                             → ไม่ match → GPT-4o-mini
                                         → confidence สูง → tag + สร้าง rule ใหม่
                                         → confidence ต่ำ → HITL Queue
```

### HITL Reviewer Flow
```
HITL item → Round-robin:
  OCR ผิด → text_verifier
  Classification ผิด → classification_verifier
  ผิดทั้งคู่ → ส่งทั้ง 2 กลุ่ม
Reviewer รับ → ตรวจ → ส่งคำตอบ → บันทึกผล + สะสมค่าตอบแทน
ไม่รับใน X นาที → ส่งคนถัดไป
```

---

## 5. Database Schema (current — session 14)

> PK ทุก table: UUID | ทุก table มี `created_at`, `updated_at` | ทุก table มี `tenant_id` ยกเว้น tenants, reviewers, reward_config

### tenants
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| name | VARCHAR(255) | ชื่อบริษัท |
| tax_id | VARCHAR(13) | UNIQUE |
| address | TEXT | ที่อยู่จดทะเบียน (header รายงานภาษี) |
| business_type | VARCHAR(20) | trading / service / construction |
| status | VARCHAR(20) | active / inactive |
| deleted_at | TIMESTAMP | soft-delete (null = ใช้งานอยู่) |
| suspended_at | TIMESTAMP | |
| suspension_reason | TEXT | |
| gdrive_folder_id | VARCHAR | |
| gdrive_folder_url | VARCHAR | link แชร์ให้ลูกค้า |

### branches
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| name | VARCHAR | |
| code | VARCHAR | UNIQUE(tenant_id, code) |
| address | TEXT | ที่อยู่สาขา |
| phone | VARCHAR(20) | |
| status | VARCHAR(20) | active / inactive |

### users
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| name | VARCHAR | ชื่อ-นามสกุล |
| email | VARCHAR | |
| phone | VARCHAR | |
| line_user_id | VARCHAR | LINE User ID |
| role | VARCHAR | admin / staff |
| status | VARCHAR | active / inactive |

### user_branches
| Column | Type |
|--------|------|
| id | UUID PK |
| user_id | UUID FK → users |
| branch_id | UUID FK → branches |

### vendors
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| tax_id | VARCHAR(13) | UNIQUE per tenant |
| name | VARCHAR | ชื่อบริษัทผู้ขาย |
| verified | BOOLEAN | ยืนยันโดย admin แล้ว? |

### invoices
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| vendor_id | UUID | FK → vendors nullable |
| file_path | VARCHAR | Path ใน MinIO |
| file_hash | VARCHAR | SHA-256 |
| status | VARCHAR(20) | pending / verified / conflict / **invalid** |
| invalid_reason | TEXT | buyer_tax_id_mismatch / buyer_branch_code_mismatch / buyer_name_mismatch / late_invoice_vat_unclaimed |
| doc_type | VARCHAR(50) | tax_invoice / receipt / invoice_billing / delivery_order |
| vat_inclusive | BOOLEAN | ราคารวม VAT แล้ว? |
| vat_rate | DECIMAL | อัตรา VAT (7%) |
| vat_math_ok | BOOLEAN | ตรวจสอบคณิตศาสตร์ VAT |
| vendor_tax_id | VARCHAR | เลขผู้เสียภาษีผู้ขาย |
| vendor_name | VARCHAR | ชื่อผู้ขาย |
| buyer_tax_id | VARCHAR | เลขผู้เสียภาษีผู้ซื้อ |
| buyer_name | VARCHAR | ชื่อผู้ซื้อ |
| buyer_branch_code | VARCHAR | รหัสสาขาผู้ซื้อ |
| invoice_doc_no | TEXT | เลขที่ใบกำกับ |
| invoice_date | TEXT | วันที่ในเอกสาร |
| invoice_year | INT | CE year |
| invoice_month | INT | |
| invoice_day | INT | |
| accounting_year | INT | รอบบัญชี ภพ.30 |
| accounting_month | INT | |
| total_before_vat | DECIMAL | ยอดก่อน VAT |
| vat_amount | DECIMAL | ยอด VAT |
| total_amount | DECIMAL | ยอดรวม |
| duplicate_of | UUID | FK → invoices nullable |
| verified_by | UUID | FK → users nullable |
| verified_at | TIMESTAMP | |
| invoice_no | SERIAL | running number |

### invoice_items
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| invoice_id | UUID | FK → invoices |
| description | TEXT | ชื่อรายการ |
| product_code | VARCHAR | |
| unit | VARCHAR | |
| quantity | DECIMAL | |
| unit_price | DECIMAL | |
| discount | DECIMAL | |
| total_price | DECIMAL | |
| asset_type | VARCHAR | asset / expense / pending |
| classified_by | VARCHAR | rule / ai / human |

### classification_rules
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| keyword | VARCHAR | คำที่ใช้ match |
| asset_type | VARCHAR | asset / expense |
| source | VARCHAR | ai / human |
| confidence | DECIMAL | |

### hitl_queue
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| invoice_item_id | UUID | FK → invoice_items |
| reason | TEXT | เหตุผลที่ค้าง |
| status | VARCHAR | pending / resolved |
| resolved_by | UUID | FK → users nullable |

### document_imports
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| user_id | UUID | FK → users |
| source_type | VARCHAR | camera / upload / zip / gdrive / onedrive |
| source_url | VARCHAR | URL ถ้ามาจาก link |
| total_files | INT | |
| processed_files | INT | |
| status | VARCHAR | pending / processing / done / failed |

### audit_logs
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| user_id | UUID | FK → users |
| action | VARCHAR | login / upload / submit / delete |
| entity_type | VARCHAR | invoice / document_import / etc |
| entity_id | UUID | |
| metadata | JSON | รายละเอียดเพิ่มเติม |
| ip_address | VARCHAR | |
| device_info | VARCHAR | |

### tenant_storage_config
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| storage_type | VARCHAR | gdrive / onedrive / both |
| gdrive_folder_id | VARCHAR | |
| gdrive_folder_url | VARCHAR | |
| onedrive_folder_id | VARCHAR | |
| onedrive_folder_url | VARCHAR | |
| owned_by | VARCHAR | tenant / us |
| billing_type | VARCHAR | included / charged |
| monthly_fee | DECIMAL | |
| status | VARCHAR | active / inactive |

### archive_policies
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| active_days | INT | เก็บใน active กี่วัน |
| archive_days | INT | เก็บใน archive กี่วัน |

### archive_logs
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| entity_type | VARCHAR | invoice / document_import |
| entity_id | UUID | |
| archived_at | TIMESTAMP | |
| archive_path | VARCHAR | Path ใน MinIO |
| status | VARCHAR | archived / restored |

### conversations
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| user_id | UUID | FK → users |
| channel | VARCHAR | line_oa / liff |
| line_user_id | VARCHAR | |
| status | VARCHAR | open / closed |

### messages
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| conversation_id | UUID | FK → conversations |
| tenant_id | UUID | FK → tenants |
| sender_type | VARCHAR | customer / admin / bot |
| sender_id | UUID | |
| message_type | VARCHAR | text / image / file / sticker |
| content | TEXT | ข้อความหรือ URL |
| metadata | JSON | LINE message ID ฯลฯ |

### reviewers
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| name | VARCHAR | |
| line_user_id | VARCHAR | |
| reviewer_type | VARCHAR | text_verifier / classification_verifier |
| status | VARCHAR | active / inactive |
| total_earned | DECIMAL | ยอดสะสมทั้งหมด |
| pending_payout | DECIMAL | รอจ่าย |

### reviewer_tasks
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| hitl_queue_id | UUID | FK → hitl_queue |
| reviewer_id | UUID | FK → reviewers |
| task_type | VARCHAR | text_verification / classification_verification |
| status | VARCHAR | sent / accepted / completed / expired |
| reward_amount | DECIMAL | |
| sent_at | TIMESTAMP | |
| accepted_at | TIMESTAMP | |
| completed_at | TIMESTAMP | |
| expired_at | TIMESTAMP | |

### reviewer_payouts
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| reviewer_id | UUID | FK → reviewers |
| amount | DECIMAL | |
| method | VARCHAR | promptpay / bank |
| account_number | VARCHAR | |
| status | VARCHAR | pending / paid |
| paid_at | TIMESTAMP | |

### reward_config
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| task_type | VARCHAR | text_verification / classification_verification |
| amount | DECIMAL | ค่าตอบแทนต่อชิ้น |
| currency | VARCHAR | THB |
| updated_by | UUID | FK → users |

---

## 6. API Endpoints

Base URL: `/api/v1`

### AUTH
```
POST   /auth/login
POST   /auth/logout
POST   /auth/refresh
```

### TENANT
```
GET    /tenants
POST   /tenants
GET    /tenants/:id
PUT    /tenants/:id
DELETE /tenants/:id          # soft delete → trash
GET    /tenants/trashed      # list trash
POST   /tenants/:id/restore  # restore จาก trash
DELETE /tenants/:id/permanent # ลบถาวร
POST   /tenants/:id/suspend
POST   /tenants/:id/unsuspend
```

### BRANCH
```
GET    /tenants/:id/branches
POST   /tenants/:id/branches
PUT    /tenants/:id/branches/:branchId
DELETE /tenants/:id/branches/:branchId
```

### USER
```
GET    /users
POST   /users
PUT    /users/:id
DELETE /users/:id
```

### DOCUMENT
```
POST   /documents/upload
POST   /documents/link
POST   /documents/zip
GET    /documents/:id/status
```

### INVOICE
```
GET    /invoices
GET    /invoices/:id
GET    /invoices/:id/items
PUT    /invoices/:id
```

### VENDOR
```
GET    /vendors
GET    /vendors/:id
POST   /vendors
PUT    /vendors/:id
POST   /vendors/:id/verify
```

### CLASSIFICATION RULES
```
GET    /rules
GET    /rules/:id
POST   /rules
PUT    /rules/:id
DELETE /rules/:id
POST   /rules/import
GET    /rules/export
POST   /rules/test
```

### HITL
```
GET    /hitl/queue
POST   /hitl/:id/resolve
POST   /hitl/:id/reject
```

### REVIEWER
```
GET    /reviewers
POST   /reviewers
PUT    /reviewers/:id
GET    /reviewers/:id/tasks
GET    /reviewers/:id/payouts
POST   /reviewers/payout
```

### CONVERSATION
```
GET    /conversations
GET    /conversations/:id/messages
POST   /conversations/:id/messages
```

### ARCHIVE
```
GET    /archive
POST   /archive/:id/restore
GET    /archive/policies
POST   /archive/policies
PUT    /archive/policies/:id
```

### STORAGE CONFIG
```
GET    /storage/config/:tenantId
POST   /storage/config
PUT    /storage/config/:tenantId
```

### REWARD CONFIG
```
GET    /reward/config
PUT    /reward/config/:id
```

### AUDIT LOG
```
GET    /audit-logs
GET    /audit-logs/:id
```

### OCR
```
POST   /ocr/extract-company   # JPG/PNG/PDF หนังสือรับรอง → CompanyData + BranchData
```

---

## 7. Migrations Applied (34 total)
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

## 8. Key Files

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
│   ├── gpt.go                     # GPT-4o-mini extraction + sendRawRequest()
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
├── Tenants.jsx          # CRUD + trash + suspend + OCR auto-fill
├── Branches.jsx         # CRUD branch + address + phone
├── Vendors.jsx          # vendor list + verify
└── ...
```

### Custom Skills
```
~/.claude/commands/
├── mem.md       # /mem — บันทึก session + sync Drive .env อัตโนมัติ
├── creds.md     # /creds — อ่าน credentials จาก Drive
├── setup.md     # /setup — new computer/project: เขียน .env + ดาวน์โหลด skills
├── popup.md     # /popup — ConfirmDialog + table action buttons
└── helpskill.md # /helpskill — list custom skills

~/.claude/scripts/
└── gdrive-update.py  # overwrite Drive file by ID (Service Account)

Drive: _claude-skills/ (1eVw_XNCf4eMn1r5FB_fQoNSeMzjMo60M) — backup ทุก skill
```

---

## 9. Invoice Status Flow
```
pending   → OCR กำลังทำงาน (auto-refresh 3s)
verified  → OCR ผ่าน + buyer valid (อาจมี invalid_reason=late_invoice เป็น warning)
conflict  → OCR cross-verify ไม่ตรง → ต้องแก้มือ
invalid   → buyer info ไม่ตรงกับ tenant/branch → ภาษีซื้อต้องห้าม ม.82/5
```

---

## 10. สิ่งที่ยังต้องทำ

### ทำได้ทันที
- [ ] ทดสอบ buyer validation — อัปโหลดใบที่ buyer_tax_id ผิด → ควรเห็น `invalid` ใน UI
- [ ] ทดสอบ OCR company extract — JPG/PNG/PDF หนังสือรับรอง → auto-fill P-01-M Create
- [ ] GPT prompt invoice: เพิ่ม `invoice_billing`/`delivery_order` ใน classification
- [x] **Google Drive overwrite script** — `~/.claude/scripts/gdrive-update.py` สร้างแล้ว, ต้องการ `gdrive-sa.json` (setup ดู section 2)

### Phase ถัดไป
- [ ] **e-Tax Invoice XML support** — parse XML มาตรฐาน RD → บันทึก invoice (ข้าม OCR) ต้องมีไฟล์ตัวอย่างก่อน
- [ ] รายงานภาษีซื้อ (ม.87/1) — export PDF/Excel พร้อม header ที่อยู่
- [ ] PDF OCR invoice (scanned PDF)
- [ ] Password reset flow
- [ ] OneDrive API integration

### Production (ยังไม่ถึงเวลา — อย่าทำ)
- Dockerfile x3, nginx+SSL, LINE OA → Target: Hetzner CX22 (~€4/เดือน)
- **อย่าสร้าง Dockerfile จนกว่าจะได้รับคำสั่ง**

---

## 11. Rules & Gotchas

**DB float:** อย่า `CASE WHEN $n != 0 THEN $n` กับ float columns — PostgreSQL infer bigint → ตัดทศนิยม ใช้ `SET col = $n` ตรงๆ เสมอ

**OCR:** Vision = อ่านข้อความ + classify เท่านั้น, GPT = extract values เท่านั้น อย่าให้ Vision extract ตัวเลข

**Backend run:** `go run ./cmd/` ไม่ใช่ `./cmd/...`

**Migration:** ต้องตั้ง `$env:MIGRATIONS_DIR` ก่อน run migrate CLI (relative path จาก `backend/` ไปไม่ถึง)

**address fields:** ไม่ใช้ใน OCR buyer validation — เก็บไว้สำหรับ header รายงานภาษีเท่านั้น

**Tenant suspend:** `checkTenantStatus` middleware ตรวจ DB ทุก request — suspended tenant โดน 403 ทันที login/refresh ก็โดนด้วย

**Popup pattern:** ทุก confirm action ใช้ `ConfirmDialog` + `confirmXxx` state + `-my-3 gap-0 py-3` (full-row click area)
- Skill: `/popup` (`~/.claude/commands/popup.md`)
- ปุ่ม "ปิดบริการ" แสดงเฉพาะ `status === 'active'`

**OCR API Keys:** เก็บใน DB (`ocr_config` table) — แต่ `/mem` sync ค่าล่าสุดขึ้น Drive `.env` ทุก session อัตโนมัติ → ถ้า DB wipe ดูค่าได้จาก Drive `.env` section `[OCR API KEYS]`

**Naming:** DB: snake_case | Go: PascalCase struct, camelCase func | API: kebab-case plural | React: PascalCase component

---

## 12. Session Status
> อัปเดตทุกครั้งที่ใช้ `/mem`

### อัพเดท: 2026-06-23 (session 19)

### ✅ Done (สิ่งที่สร้างแล้ว)
- Infrastructure: Docker Compose (PostgreSQL/Redis/MinIO), **34 migrations** ครบ
- Backend: 70+ endpoints, OCR dual-engine, Asynq queue, HITL, reviewer, audit, archive, LINE webhook
- Migration runner: `db/migrate.go` + `cmd/migrate/` + auto-migrate on startup
- Admin UI: InvoiceDetail (full rewrite), VerificationWizard, Invoices, Tenants, Branches, Vendors, Settings
- LIFF: Login, branch select, upload, status, conversation
- **Session 11**: Vendor registry, invoice date parts, accounting period, duplicate detection
- **Session 12**: Thai tax law validation layer, address fields
- **Session 13**: Dev labels, P-01-M unified form, OCR company extract (JPG/PNG)
- **Session 14**: Tenant soft-delete (trash), suspend/unsuspend, UX modal system
- **Session 15**: PDF OCR company, Confirm Popup pattern, Tenant suspend enforcement
- **Session 16**: Credentials system (Google Drive), custom skills (/creds /mem /popup /helpskill)
- **Session 17**: Merge handoff.md → CLAUDE.md (single source of truth), update /mem skill
- **Session 18**: gdrive-update.py script, /mem auto-sync .env
- **Session 19**: /setup skill, skills backup to Drive _claude-skills/, Drive .env ครบทุก credential

### ✅ Session 19 — /setup skill + Skills Backup to Drive (2026-06-23)

**Skills backup system:**
- สร้าง `/setup` skill — new computer/project: อ่าน Drive → เขียน .env + ดาวน์โหลด skills อัตโนมัติ
- Upload skills ทั้งหมดขึ้น Drive `_claude-skills/` (folder ID: `1eVw_XNCf4eMn1r5FB_fQoNSeMzjMo60M`)
- `/mem` step 3b: auto-sync `.env` ทุก session — query DB `ocr_config` + อ่าน local .env → upload ทับ Drive
- Drive `.env` ครบทุกค่า: DOCKER + BACKEND + OPENAI_API_KEY + GCV_API_KEY + ADMIN UI
- `/creds` ใช้ Drive `.env` เป็น single source สำหรับทุก credential

**Flow เครื่องใหม่:** `git clone` → `/setup` → พร้อมรัน (ไม่ต้องกรอก credential ใดๆ)

### ✅ Session 18 — Google Drive Overwrite Script (2026-06-23)

**GDrive overwrite script:**
- สร้าง `~/.claude/scripts/gdrive-update.py` — overwrite Drive file by ID ด้วย Service Account
- Install `google-api-python-client`, `google-auth` ลง Python env แล้ว
- อัปเดต `/mem` skill: step 3b sync .env อัตโนมัติ
- ขั้นตอนถัดไป: สร้าง `gdrive-sa.json` (Service Account key) + share folder

### ✅ Session 17 — Unified Memory File (2026-06-23)

**Merge handoff.md → CLAUDE.md:**
- ลบ handoff.md ออก — CLAUDE.md เป็น single source of truth ทั้ง AI และ Drive
- `/mem` skill อัปเดต CLAUDE.md เพียงไฟล์เดียว + upload ขึ้น Drive แทน handoff.md
- CLAUDE.md restructure: full schema + full API endpoints กลับเข้า, ตัดแค่ folder tree / MVP checklist / naming convention

### 🔑 OCR Architecture (final — ห้ามเปลี่ยน)
- Vision: อ่าน Thai text + classify doc_type/vat_inclusive (ไม่ extract ตัวเลข)
- GPT: รับ text + VISION HINT → extract ทุก field (sole authority)
- Key files: `ocr/vision.go`, `ocr/gpt.go`, `ocr/service.go`, `ocr/crossverify.go`, `ocr/company.go`

### 🟡 ถัดไป (ทำได้เลย)
- ทดสอบ buyer validation: อัปโหลดใบที่ buyer_tax_id ผิด → ควรเห็น status=invalid ใน UI
- GPT prompt invoice: เพิ่ม `invoice_billing`/`delivery_order` ใน classification prompt
- **Setup gdrive-sa.json** — สร้าง Service Account → download JSON → share `_claude-skills/` + `tax-ocr/` folders → ทดสอบ overwrite script

### 🔵 Phase ถัดไป
- OneDrive API, PDF OCR (invoice), Password reset, รายงานภาษีซื้อ (ม.87/1), e-Tax XML

### Production Plan (ยังไม่ถึงเวลา)
- Target: Hetzner CX22 (~€4/เดือน), Docker Compose
- ต้องทำก่อน: Dockerfile x3, nginx+SSL, LINE OA
- **อย่าสร้าง Dockerfile จนกว่าจะได้รับคำสั่ง**
