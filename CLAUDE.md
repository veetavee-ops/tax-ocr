# CLAUDE.md — Tax OCR System (Memory File)
> อ่านไฟล์นี้ทุกครั้งก่อนเริ่ม session ใหม่

---

## 1. Project Overview

**ชื่อโปรเจกต์:** Tax OCR System  
**วัตถุประสงค์:** ระบบรับใบกำกับภาษีจากลูกค้าผ่าน LINE LIFF, ประมวลผลด้วย OCR Dual-Engine, แยกประเภทรายการ (สินทรัพย์/ค่าใช้จ่าย) ด้วย Hybrid Classification และจัดการ HITL ด้วยระบบ Crowdsourced Reviewer

**MVP Scope:**
- LINE LIFF สำหรับลูกค้าส่งเอกสาร
- Admin UI สำหรับทีมบัญชีจัดการหลังบ้าน
- OCR ด้วย Cloud AI (GPT-4o-mini + Google Cloud Vision)
- Hybrid Classification (Rule → AI → HITL)
- Self-learning Rule Engine
- Crowdsourced Reviewer System (2 กลุ่ม)
- Multi-tenant + Multi-branch
- Audit Log ทุก action
- Data Lifecycle (Active → Archive)

---

## 2. Tech Stack

### FRONTEND
- LINE LIFF — React + Vite (User UI)
- Admin UI — React + Vite + Tailwind

### BACKEND
- Go (Golang)
- Asynq + Redis (Queue)

### DATABASE
- PostgreSQL (Primary Database)
- MinIO (Object Storage)

### EXTERNAL SERVICES
- GPT-4o-mini (Structure Extraction + Classification)
- Google Cloud Vision (OCR Text Reading)
- Google Drive API (Customer Storage)
- OneDrive API (Customer Storage)
- LINE Messaging API + LIFF SDK

### INFRASTRUCTURE
- Docker Compose (Local Dev)

---

## 3. Folder Structure

```
/tax-ocr/
├── CLAUDE.md
├── frontend/
│   ├── liff/                        # LINE LIFF (User UI)
│   │   ├── src/
│   │   │   ├── components/
│   │   │   ├── pages/
│   │   │   └── main.jsx
│   │   └── package.json
│   └── admin/                       # Admin UI
│       ├── src/
│       │   ├── components/
│       │   ├── pages/
│       │   └── main.jsx
│       └── package.json
├── backend/
│   ├── cmd/
│   │   └── main.go
│   ├── internal/
│   │   ├── api/                     # HTTP handlers
│   │   ├── queue/                   # Asynq workers
│   │   ├── ocr/                     # OCR logic
│   │   ├── classify/                # Asset classification
│   │   └── db/                      # DB queries
│   ├── pkg/                         # Shared utilities
│   └── go.mod
├── database/
│   └── migrations/                  # SQL migration files
└── infrastructure/
    ├── docker-compose.yml
    └── .env.example
```

---

## 4. Database Schema

> Primary Key ทุก table ใช้ UUID  
> ทุก table มี `created_at`, `updated_at`  
> ทุก table มี `tenant_id` ยกเว้น tenants

### tenants
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| name | string | ชื่อบริษัท |
| tax_id | string | เลขผู้เสียภาษี 13 หลัก |
| status | string | active / inactive |
| gdrive_folder_id | string | Google Drive Folder ID |
| gdrive_folder_url | string | Link แชร์ให้ลูกค้า |

### branches
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| name | string | ชื่อสาขา |
| code | string | รหัสสาขา |
| status | string | active / inactive |

### users
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| name | string | ชื่อ-นามสกุล |
| email | string | |
| phone | string | |
| line_user_id | string | LINE User ID |
| role | string | admin / staff |
| status | string | active / inactive |

### user_branches
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| user_id | UUID | FK → users |
| branch_id | UUID | FK → branches |

### invoices
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| file_path | string | Path ใน MinIO |
| file_hash | string | SHA-256 |
| vendor_tax_id | string | เลขผู้เสียภาษีผู้ขาย |
| total_before_vat | decimal | ยอดก่อน VAT |
| vat_amount | decimal | ยอด VAT |
| total_amount | decimal | ยอดรวม |
| status | string | pending / verified / conflict |

### invoice_items
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| invoice_id | UUID | FK → invoices |
| description | string | ชื่อรายการ |
| quantity | decimal | จำนวน |
| unit_price | decimal | ราคาต่อหน่วย |
| total_price | decimal | ราคารวม |
| asset_type | string | asset / expense / pending |
| classified_by | string | rule / ai / human |

### classification_rules
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| keyword | string | คำที่ใช้ match |
| asset_type | string | asset / expense |
| source | string | ai / human |
| confidence | decimal | ความมั่นใจ |

### hitl_queue
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| invoice_item_id | UUID | FK → invoice_items |
| reason | string | เหตุผลที่ค้าง |
| status | string | pending / resolved |
| resolved_by | UUID | FK → users |

### document_imports
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| user_id | UUID | FK → users |
| source_type | string | camera / upload / zip / gdrive / onedrive |
| source_url | string | URL ถ้ามาจาก link |
| total_files | int | จำนวนไฟล์ทั้งหมด |
| processed_files | int | ประมวลผลแล้ว |
| status | string | pending / processing / done / failed |

### audit_logs
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| user_id | UUID | FK → users |
| action | string | login / upload / submit / delete |
| entity_type | string | invoice / document_import / etc |
| entity_id | UUID | FK ไปหา record ที่เกี่ยวข้อง |
| metadata | JSON | รายละเอียดเพิ่มเติม |
| ip_address | string | |
| device_info | string | |

### tenant_storage_config
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| storage_type | string | gdrive / onedrive / both |
| gdrive_folder_id | string | |
| gdrive_folder_url | string | |
| onedrive_folder_id | string | |
| onedrive_folder_url | string | |
| owned_by | string | tenant / us |
| billing_type | string | included / charged |
| monthly_fee | decimal | ถ้า charged |
| status | string | active / inactive |

### archive_policies
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| active_days | int | เก็บใน active กี่วัน |
| archive_days | int | เก็บใน archive กี่วัน |

### archive_logs
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| entity_type | string | invoice / document_import |
| entity_id | UUID | |
| archived_at | timestamp | |
| archive_path | string | Path ใน MinIO |
| status | string | archived / restored |

### conversations
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| tenant_id | UUID | FK → tenants |
| branch_id | UUID | FK → branches |
| user_id | UUID | FK → users |
| channel | string | line_oa / liff |
| line_user_id | string | LINE User ID |
| status | string | open / closed |

### messages
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| conversation_id | UUID | FK → conversations |
| tenant_id | UUID | FK → tenants |
| sender_type | string | customer / admin / bot |
| sender_id | UUID | user_id หรือ admin_id |
| message_type | string | text / image / file / sticker |
| content | string | ข้อความหรือ URL |
| metadata | JSON | LINE message ID ฯลฯ |

### reviewers
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| name | string | |
| line_user_id | string | LINE User ID |
| reviewer_type | string | text_verifier / classification_verifier |
| status | string | active / inactive |
| total_earned | decimal | ยอดสะสมทั้งหมด |
| pending_payout | decimal | รอจ่าย |

### reviewer_tasks
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| hitl_queue_id | UUID | FK → hitl_queue |
| reviewer_id | UUID | FK → reviewers |
| task_type | string | text_verification / classification_verification |
| status | string | sent / accepted / completed / expired |
| reward_amount | decimal | ค่าตอบแทน |
| sent_at | timestamp | |
| accepted_at | timestamp | |
| completed_at | timestamp | |
| expired_at | timestamp | |

### reviewer_payouts
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| reviewer_id | UUID | FK → reviewers |
| amount | decimal | |
| method | string | promptpay / bank |
| account_number | string | |
| status | string | pending / paid |
| paid_at | timestamp | |

### reward_config
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | PK |
| task_type | string | text_verification / classification_verification |
| amount | decimal | ค่าตอบแทนต่อชิ้น |
| currency | string | THB |
| updated_by | UUID | FK → users |

---

## 5. API Endpoints

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
```

### BRANCH
```
GET    /tenants/:id/branches
POST   /tenants/:id/branches
PUT    /tenants/:id/branches/:branchId
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

---

## 6. Business Logic

### OCR Flow
```
1. ลูกค้าส่งเอกสาร (camera / upload / zip / gdrive / onedrive)
2. Backend รับไฟล์ → บันทึกลง MinIO → โยนเข้า Asynq Queue
3. Worker ดึงงานจาก Queue
4. OCR Dual-Engine พร้อมกัน
   - Engine 1: GPT-4o-mini (ดึงโครงสร้าง)
   - Engine 2: Google Cloud Vision (อ่านตัวอักษร)
5. Cross-verify Header/Footer
   - เลขผู้เสียภาษี 13 หลัก
   - ยอดก่อน VAT
   - ยอด VAT
   - ยอดรวม
6. ตรงกัน → ทำ SHA-256 Hash → บันทึกลง PostgreSQL
7. ไม่ตรงกัน → เข้า HITL Queue → แจ้งเตือน Reviewer
```

### Classification Flow (Hybrid)
```
1. แต่ละ Line Item ผ่าน Rule-based ก่อน
2. Match Rule → Tag asset/expense → จบ
3. ไม่ Match → ยิง GPT-4o-mini
4. Confidence สูง → Tag อัตโนมัติ + สร้าง Rule ใหม่
5. Confidence ต่ำ → เข้า HITL Queue
6. Admin/Reviewer ตัดสิน → สร้าง Rule ใหม่อัตโนมัติ
```

### HITL Reviewer Flow
```
1. HITL item เข้า Queue
2. ระบบส่งให้ Reviewer แบบ Round-robin
   - OCR ผิด → ส่งให้ text_verifier
   - Classification ผิด → ส่งให้ classification_verifier
   - ผิดทั้งคู่ → ส่งให้ทั้ง 2 กลุ่ม
3. Reviewer รับงาน → ตรวจสอบ → ส่งคำตอบ
4. ระบบบันทึกผล → สะสมค่าตอบแทน
5. ถ้าไม่รับใน X นาที → ส่งคนถัดไป
```

### Data Lifecycle
```
Active   → PostgreSQL (ตาม active_days ของ tenant)
Archive  → MinIO Cold Storage (ตาม archive_days ของ tenant)
หมายเหตุ: ไม่มี Auto Delete ทุก record เก็บตลอด
```

---

## 7. UI Screens

### LINE LIFF (User UI)
1. หน้า Login — LINE Login button
2. หน้าเลือก Branch — Dropdown/List (ข้ามถ้ามีสาขาเดียว)
3. หน้าส่งเอกสาร — Camera / Upload / ZIP / GDrive / OneDrive
4. หน้าติดตามสถานะ — รายการเอกสารที่ส่งมา + status
5. หน้าประวัติการสนทนา — Chat history กับทีม

### Admin UI
1. Dashboard — สถิติภาพรวม, กราฟ, Queue status
2. Tenant Management — CRUD tenant
3. Branch Management — CRUD branch
4. User Management — CRUD user + assign branch
5. Invoice List — ดูใบกำกับภาษีทั้งหมด + detail
6. HITL Queue — จัดการรายการรอตรวจสอบ + Reviewer system
7. Classification Rules — จัดการ Rule list + test rule
8. Conversation — Chat history + ตอบลูกค้า
9. Storage Config — จัดการ GDrive/OneDrive ต่อ tenant
10. Archive — ดู/restore archive + จัดการ policy
11. Audit Log — ประวัติทุก action
12. Settings — Reward config, OCR config, LINE config

---

## 8. Naming Convention

### Database
- Tables: `snake_case` พหูพจน์ เช่น `invoice_items`
- Columns: `snake_case` เช่น `tenant_id`, `created_at`
- Primary Key: `id` (UUID)
- Foreign Key: `{table_singular}_id` เช่น `tenant_id`
- Timestamps: `created_at`, `updated_at`

### Backend (Go)
- Package: `lowercase` เช่น `package api`
- Struct: `PascalCase` เช่น `Invoice`
- Exported Function: `PascalCase` เช่น `GetInvoice()`
- Internal Function: `camelCase` เช่น `processQueue()`
- Variable: `camelCase` เช่น `tenantID`
- File: `snake_case` เช่น `invoice_handler.go`
- Constants: `UPPER_SNAKE_CASE` เช่น `MAX_RETRY`

### API Endpoints
- Resource: `kebab-case` พหูพจน์ เช่น `/invoice-items`
- Version: `/api/v1/...`
- Pattern: `/api/v1/{resource}/{id}/{sub-resource}`

### Frontend (React)
- Component: `PascalCase` เช่น `InvoiceList`
- Hook: `camelCase` prefix `use` เช่น `useInvoice()`
- File: `PascalCase` เช่น `InvoiceList.jsx`
- CSS Class: `kebab-case` เช่น `invoice-table`

### Environment Variables
- `UPPER_SNAKE_CASE` เช่น `DB_HOST`, `GPT_API_KEY`

---

## 9. MVP Scope

### ทำใน MVP นี้
- [ ] Docker Compose setup (PostgreSQL, Redis, MinIO)
- [ ] Database migrations ทุก table
- [ ] Backend Go: Auth, Tenant, Branch, User APIs
- [ ] Backend Go: Document upload + Queue
- [ ] OCR Pipeline: GPT-4o-mini + Google Cloud Vision
- [ ] Cross-verify logic
- [ ] Classification: Rule-based + AI + Self-learning
- [ ] HITL Queue + Reviewer assignment
- [ ] LINE LIFF: Login, เลือก Branch, ส่งเอกสาร
- [ ] Admin UI: ทุกหน้าตาม spec
- [ ] Audit Log
- [ ] Archive system

### ไม่ทำใน MVP (Phase ถัดไป)
- Local AI (Ollama + PaddleOCR)
- Auto-scale / Horizontal scaling
- Mobile App แยก
- Advanced Analytics

---

*อัพเดทล่าสุด: สร้างจากการออกแบบร่วมกับ Architect*

---

## 10. Session Status
> **สำหรับ AI:** section นี้คือ memory ข้ามสรรหา อัปเดตทุกครั้งที่ผู้ใช้สั่ง "mem" หรือ "บันทึก session" หรือ "save"
> อัปเดต in-place — ไม่ต้องสร้างไฟล์ใหม่

### วิธีรัน Local Dev
```powershell
cd e:\tax-ocr\infrastructure && docker compose up -d
cd e:\tax-ocr\backend        && go run ./cmd/          # port 8080 (auto-migrate)
cd e:\tax-ocr\frontend\admin && npm run dev            # port 3000
cd e:\tax-ocr\frontend\liff  && npm run dev            # port 5174
```
- Login: veetavee@gmail.com / test1234
- PostgreSQL host port: **5433**, Redis: **6380**
- DB shell: `docker exec -it tax-ocr-postgres psql -U tax_ocr -d tax_ocr`
- ⚠️ รัน backend: `go run ./cmd/` ไม่ใช่ `./cmd/...` (มี 2 packages แล้ว)

### Migration
```powershell
go run ./cmd/migrate/ -stamp   # DB เดิมที่ยังไม่มี schema_migrations (ทำครั้งเดียว)
go run ./cmd/migrate/          # apply migrations ที่ยังไม่ได้ run
```

### อัพเดท: 2026-06-23 (session 16)

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

### ✅ Session 16 — Credentials System + Custom Skills (2026-06-23)

**Credentials System (Google Drive):**
- credentials ทั้งหมดเก็บใน Google Drive `tax-ocr/.env` — ไม่ขึ้น git
- Drive root: `https://drive.google.com/drive/folders/17BDc1uAofvv5irAeaf4pgVRxMV7AqP2l`
- `backend/.env` สร้างแล้ว (JWT_SECRET + infra config)
- OCR API keys (OPENAI, GCV) เก็บใน DB `ocr_config` table — ดึงจาก DB ผ่าน `buildOCRConfigFromDB()`
- handoff.md section 1 เปลี่ยนจาก hardcode → Drive link

**Custom Skills (`~/.claude/commands/`):**
- `/creds` — อ่าน credentials จาก Drive แทนการถามผู้ใช้
- `/mem` — บันทึก session: อัปเดต handoff.md (10 sections) + CLAUDE.md + อัปโหลด Drive + ถาม commit
- `/popup` — ConfirmDialog + Table action buttons pattern (เดิมชื่อ `/ui-modal`)
- `/helpskill` — แสดง custom skills ทั้งหมด

### ✅ Session 15 — PDF OCR + Popup UX + Suspend Enforcement (2026-06-23)

**PDF OCR Company:**
- `github.com/ledongthuc/pdf` — extract text จาก digital PDF โดยตรง (ไม่ต้อง Vision)
- `extractTextFromPDF()` ใน `ocr/company.go` — ใช้ `bytes.NewReader` ไม่ต้อง temp file
- `ExtractCompanyInfo()` แยก path: PDF → text → GPT / Image → Vision → GPT
- ถ้า PDF ไม่มีข้อความ (scanned) → error ชัดเจน "กรุณาใช้ไฟล์รูปแทน"
- Frontend: file input รับ `image/jpeg,image/png,application/pdf` + label อัปเดต

**Confirm Popup UX (global pattern):**
- ลบ `SuspendDialog` ออก — ทุก action ใช้ `ConfirmDialog` pattern เดียวกันหมด
- State pattern: `confirmXxx` / `setConfirmXxx` / `doXxx()` (ไม่รับ param)
- Table action buttons: `-my-3 gap-0` wrapper + `py-3` = full-row click area
- ปุ่ม "ปิดบริการ" แสดงเฉพาะ `status === 'active'` เท่านั้น
- Global skill เปลี่ยนชื่อ: `/ui-modal` → `/popup`

**Tenant Suspend Enforcement:**
- `checkTenantStatus` middleware — ตรวจ DB ทุก request ใน `/api/v1/` → 403 ถ้า suspended
- `login` / `refresh` — block ถ้า tenant suspended → ต่ออายุ session ไม่ได้
- ลำดับ middleware: `authMiddleware → checkTenantStatus → auditMiddleware → handler`

### ✅ Session 14 — Tenant Trash + Suspend + Modal UX System (2026-06-23)

**Tenant Soft Delete (Trash):**
- Migration 034: เพิ่ม `deleted_at`, `suspended_at`, `suspension_reason` ใน tenants
- `DeleteTenant` → soft delete, `ListTrashedTenants`, `RestoreTenant`, `PermanentDeleteTenant`
- Admin UI: tab "ใช้งานอยู่" / "ถังขยะ" + ปุ่มเรียกคืน + ลบถาวร

**Tenant Suspend:**
- `SuspendTenant(reason)`, `UnsuspendTenant` — backend + routes
- ปุ่ม "ปิดบริการ" / "เปิดบริการ" ใน active tab
- `StatusBadge` เพิ่ม `suspended` = orange

**Modal UX System (global — ทุกโปรเจกต์):**
- `Modal.jsx`: ESC = close, click backdrop = close, Arrow ←→ = เลื่อน focus ระหว่างปุ่ม, `hideClose` prop
- `useDblClickProtect(isFocused)` hook — คลิก mouse ขณะยังไม่ focus = set focus เท่านั้น ต้องคลิกอีกครั้งจึงทำงาน
- `ConfirmDialog` component — focus-swap color, double-click protect ทั้ง 2 ปุ่ม
- Global skill: `~/.claude/commands/popup.md`

### ✅ Session 13 — Dev Labels + Tenant UX + OCR Company (2026-06-22)

**Dev Labels (ลบตอน production):**
- `DevLabel` component — badge `P-00`–`P-13` มุมล่างซ้ายทุกหน้า (ใน Layout.jsx)
- `Modal.jsx`: `devLabel` prop — badge ใน header ทุก modal `P-01-M`–`P-12R-M`
- `ImageViewer`, `VerificationWizard`: inline badge `P-05-M2`, `P-05-W`
- ลบ production: ลบ `<DevLabel />` + import ใน Layout.jsx, ลบ `devLabel` prop ใน Modal.jsx

**P-01-M Tenant Modal — unified form:**
- Create+Edit ใช้ form เดียวกัน: ID(edit-only/read-only), tax_id(create=editable/edit=read-only), name, business_type, address, status
- Backend `CreateTenant()` เพิ่ม `address` param
- `POST /tenants` รับ address + status แล้ว

**OCR Company Extract (ใหม่):**
- `POST /api/v1/ocr/extract-company` — JPG/PNG → Vision → GPT / PDF → text → GPT
- `ocr/company.go`: `CompanyData` + `BranchData` structs, `extractTextFromPDF()`
- `gpt.go`: เพิ่ม `sendRawRequest()` helper (generic, คืน raw bytes)
- P-01-M Create: ปุ่ม "📷 อ่านเอกสาร" → auto-fill ทุก field + preview สาขา → สร้าง branch อัตโนมัติหลัง submit

### 🔑 กฎ DB: อย่า CASE WHEN กับ float columns
> `CASE WHEN $n != 0 THEN $n` — PostgreSQL infer type จาก integer literal `0` → cast bigint → ตัดทศนิยม
> **กฎ**: financial amount columns ใช้ direct `SET col = $n` เสมอ

### 🔑 OCR Architecture (final — ห้ามเปลี่ยน)
- Vision: อ่าน Thai text + classify doc_type/vat_inclusive (ไม่ extract ตัวเลข)
- GPT: รับ text + VISION HINT → extract ทุก field (sole authority)
- Key files: `ocr/vision.go`, `ocr/gpt.go`, `ocr/service.go`, `ocr/crossverify.go`, `ocr/company.go`

### 🟡 ถัดไป (ทำได้เลย)
- **ทดสอบ** buyer validation: อัปโหลดใบที่ buyer_tax_id ผิด → ควรเห็น status=invalid ใน UI
- **GPT prompt invoice**: เพิ่ม `invoice_billing`/`delivery_order` ใน classification prompt

### 🔵 Phase ถัดไป
- OneDrive API, PDF OCR (invoice), Password reset, รายงานภาษีซื้อ (ม.87/1)

### Production Plan (ยังไม่ถึงเวลา)
- Target: Hetzner CX22 (~€4/เดือน), Docker Compose
- ต้องทำก่อน: Dockerfile x3, nginx+SSL, LINE OA
- **อย่าสร้าง Dockerfile จนกว่าจะได้รับคำสั่ง**
