# Simple Distributed Job Queue Simulation
a simple job queue system built by golang

---

## Part 1 — Job Queue Backend (GraphQL)

### Requirement:
-> Go >= 1.20

### How to Run:
-> on VsCode -> Run(Pick Go-Debug)
-> open localhost:58579/graphiql
-> should be look like this
![Graphiql](./docs/graphiql.jpg)

### Requirements
make sure you complete all the function stated at first you load graphiql page
1. SimultaneousCreateJob -> build a simultaneous job
2. SimulateUnstableJob -> special case for task "unstable-job" fails twice before passing
3. GetAllJobs -> get all jobs that already registered
4. GetJobById -> get job by the id that been created by enqueue job
5. GetAllJobStatus -> get the stats of all jobs that been processed

### Evaluation Criteria:
* **Correctness**: Job creation, execution, status updates are accurate.
* **Concurrency Safety**: Multiple jobs created/processed at once → no race, no corruption.
* **Idempotency Handling**: Same job/task with same ID or token → doesn't process twice.
* **Retry Logic**: Failing job retries up to N times with delay.
* **In-memory Safety**: Maps/lists used safely under concurrent access.
* **Code Quality**: Idiomatic Go, good naming, clean package layout.
* **Clean Architecture**: Separation of domain, repository, resolver, GraphQL models.
* **Performance Awareness**: Handles 50–100 concurrent jobs without crash or slowdown.
* **Logging and Debugging**: Logs meaningful events.
* **Graceful Failure Handling**: No panics; job failure doesn't crash system.

---

## Part 2 — HTMX Dashboard

### Overview
Build a server-rendered dashboard at `/jobqueue/dashboard` using [HTMX](https://htmx.org/). The backend handler lives in `delivery/htmx/`; the HTML templates live in `web/htmx/`. No JavaScript framework is needed — HTMX attributes drive all partial page updates.

### Routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/jobqueue/dashboard` | Full HTML page shell |
| `GET` | `/jobqueue/dashboard/message` | Initial HTMX fragment entry point |

> Add more fragment endpoints under `/jobqueue/dashboard/...` as each section is built out.

---

### Backend (Go — `delivery/htmx/`)

Architecture rules:
- Handlers live in `delivery/htmx/` — the delivery layer only.
- Handlers that need data must call the service layer through the interfaces in `interface/`.
- No business logic inside handlers; handlers only bridge HTTP ↔ service.

#### Fragment Endpoints to Implement

| Endpoint | Description |
|----------|-------------|
| `POST /jobqueue/dashboard/jobs/create` | Runs `SimultaneousCreateJob` with Job1/Job2/Job3 from the form, returns updated job list fragment |
| `POST /jobqueue/dashboard/jobs/unstable` | Runs `SimulateUnstableJob`, returns updated job list fragment |
| `GET /jobqueue/dashboard/status` | Returns the status summary fragment (polled every 2 s) |
| `GET /jobqueue/dashboard/jobs` | Returns the full jobs table fragment (polled every 2 s) |
| `GET /jobqueue/dashboard/jobs/:id` | Returns the job detail fragment for a single job |

---

### Frontend (HTMX — `web/htmx/`)

#### 1. Action Bar
Three trigger buttons at the top of the page:

| Button | Fires | Notes |
|--------|-------|-------|
| **Create 3 Jobs** | `POST /jobqueue/dashboard/jobs/create` | Reads Job1/Job2/Job3 from the Variables Form |
| **Create Unstable Job** | `POST /jobqueue/dashboard/jobs/unstable` | Hard-coded `task: "unstable-job"` |
| **Refresh** *(optional)* | Re-polls `/jobqueue/dashboard/status` and `/jobqueue/dashboard/jobs` | Manual trigger |

#### 2. Variables Form
Inline form whose values feed the **Create 3 Jobs** action. Defaults must match `web/variables.json`:

| Field | Default |
|-------|---------|
| Job1 | `JobTest1` |
| Job2 | `JobTest2` |
| Job3 | `JobTest3` |

#### 3. Status Summary
Live badge/card row — auto-polled every 2 seconds via `hx-trigger="every 2s"` targeting `#status-summary`:

- **Pending** / **Running** / **Failed** / **Completed**

#### 4. Jobs Table
Table auto-polled every 2 seconds targeting `#jobs-table`:

| Column | Field |
|--------|-------|
| ID | `job.id` |
| Task | `job.task` |
| Status | `job.status` |
| Attempts | `job.attempts` |
| View | Button → loads Job Detail Panel for that row's ID |

#### 5. Job Detail Panel
Panel or side section targeting `#job-detail`:
- Triggered by clicking **View** in any table row, or by typing a job ID into a search input.
- Displays: `id`, `task`, `status`, `attempts`.
- Swapped in via `hx-get="/jobqueue/dashboard/jobs/:id"` without reloading the rest of the page.

### How to Access
1. Start the server (see **How to Run** above).
2. Open `http://localhost:58579/jobqueue/dashboard`.
3. Status summary and job list auto-refresh every 2 seconds via HTMX polling.

# Good Luck Guys

---

# Catatan Pengerjaan (Personal Notes)

> Catatan pribadi saya waktu ngerjain technical test ini. Ditulis pake bahasa sendiri biar gampang di-revisit kalau lupa. Bukan dokumentasi resmi, cuma brain dump.

## Context Singkat

Technical test-nya minta bikin **distributed job queue simulation** pake Go + GraphQL buat backend-nya, terus ada dashboard HTMX buat monitoring. Project udah punya kerangka Clean Architecture (handler → service → repository) tapi大部分 logic-nya masih stub kosong — jadi kerjaan utama saya adalah ngisi semua itu sampai jadi functional.

## Arsitektur yang Saya Pahami Dulu

Sebelum nulis kode, saya baca-baca dulu struktur project. Paham ada 3 layer:

- **Handler** (di `delivery/`) — cuma nerima HTTP request, validasi tipis, lempar ke service. Nggak boleh ada business logic di sini.
- **Service** (di `service/`) — aturan bisnis: retry, idempotency, special case unstable-job, dll.
- **Repository** (di `repository/`) — simpan/ambil data. Di project ini pake in-memory map + `sync.RWMutex` biar thread-safe.

Kontrak komunikasi antar layer ditulis di `interface/job.go`. Repository udah implements interface-nya, tinggal service yang perlu diisi.

## Part 1 — Backend Job Queue (GraphQL)

### Step Awal: Fix Bug di Interface

Hal pertama yang saya notice: tanda tangan `GetAllJobs` di `interface/job.go` salah. Balikannya `entity.Job` (satu doang), padahal harusnya `[]*entity.Job` karena `FindAll` di repository udah return list. Bug kecil tapi krusial — kalau dibiarin, semua yang manggil `GetAllJobs` bakal bingung.

### Yang Saya Bangun di Service

1. **Enqueue** — bikin UUID baru, set status `pending`, simpan ke repo, terus **langsung balikin ID**. Proses job-nya saya jalanin di **goroutine terpisah** pake `go q.processJob(id)` biar response GraphQL nggak nunggu. Penting: goroutine-nya pake `context.Background()` karena context dari request GraphQL bakal ke-cancel begitu response balik.

2. **processJob** — di sini state machine-nya. Loop sampai max 3 attempts:
   - Set status `running`, save
   - Sleep 2 detik (simulasi kerja)
   - Cek `executeJob(job)`:
     - Kalau task-nya `"unstable-job"` dan attempt ≤ 2 → return error (simulasi gagal)
     - Task lain saya kasih 10% chance gagal random biar retry logic-nya keliatan jalan
   - Kalau gagal dan masih ada sisa attempt → balik ke `pending`, sleep 1 detik, retry
   - Kalau gagal dan udah habis → set `failed`
   - Kalau sukses → set `completed`

3. **Idempotency** — pakai map `taskLocks[taskName] -> jobID` yang dijaga `sync.RWMutex`. Kalau ada request `Enqueue(task: "send-email")` sementara job dengan nama yang sama masih `pending`/`running`, balikin ID yang udah ada. Setelah selesai, lock-nya di-release. Hal ini supaya client nggak sengaja bikin duplicate job kalau dia tekan tombol 2x cepat.

4. **Retry logic** — saya hardcode `maxAttempts = 3` dan `retryDelay = 1 * time.Second`. Konfigurasi ini bisa dipindah ke config kalau mau lebih proper, tapi untuk test ini cukup.

### Yang Saya Pelajari di Part Ini

- **`sync.RWMutex`** lebih cocok dari `sync.Mutex` buat kasus baca-banyak-tulis-sedikit. Dashboard HTMX polling tiap 2 detik artinya banyak read, jadi `RLock()` bikin baca bisa paralel, ngga saling tunggu.
- **Goroutine + context** — request-scoped context bahaya buat background work karena di-cancel begitu response balik. Solusinya pake `context.Background()` atau derive context baru.
- **graph-gophers/graphql-go** — argumen di-resolve via struct dengan field name matching sama GraphQL schema. Kalau salah nama, error-nya agak cryptic tapi kelihatan di console.

### Kendala yang Saya Temuin

- Awalnya saya taruh field sama method namanya `ID` di resolver — compile error karena Go bingung. Saya rename field-nya jadi `JobID`, method `ID()` tetap ada buat dipanggil GraphQL.
- GraphQL return `*resolver.JobResolver` itu pointer ke struct. Method `Status()`, `Task()`, dll di resolver itu sebenernya query ulang ke service. Bisa pake dataloader buat optimasi, tapi karena jumlah job masih kecil, query langsung udah cukup.

## Part 2 — HTMX Dashboard

### Yang Saya Ganti

- `web/htmx/hello.html` saya hapus, ganti `web/htmx/dashboard.html` yang isinya lengkap: action bar (4 tombol), variables form (Job1/Job2/Job3), status badges, jobs table, job detail panel.
- `delivery/htmx/handler.go` saya rewrite dari `HelloHandler` jadi `DashboardHandler` dengan 7 method:
  - `Page` — serve full HTML shell
  - `Message` — fragment welcome (entry point sesuai Readme)
  - `CreateJobs` / `CreateUnstable` — POST handler
  - `StatusFragment` / `JobsFragment` / `JobDetailFragment` — GET handler buat polling
  - `JobDetailByQuery` — handler khusus buat "Load by ID" button

### Auto-Refresh 2 Detik

Pakai `hx-trigger="every 2s"` di element yang mau di-poll. Tiap fragment yang dibalikin juga ikut menyertakan atribut polling itu sendiri, jadi HTMX otomatis lanjut polling setelah swap. Saya juga sertakan polling di tombol Refresh manual buat backup.

### Polling vs Streaming

Saya pilih polling 2 detik bukan WebSocket/SSE. Alasan:
- Implementasi jauh lebih simpel
- 2 detik cukup responsive buat monitoring job queue
- Dashboard ngga butuh real-time sampai milisecond

Trade-off: ada delay 2 detik sebelum status berubah kelihatan. Untuk use case "klik tombol → liat status berubah" masih oke.

### Yang Saya Pelajari di Part Ini

- **HTMX** — atribut di HTML yang auto-trigger HTTP call. `hx-target` + `hx-swap="outerHTML"` buat replace isi element. `hx-include` buat kirim data form. `hx-vals='js:{...}'` buat ambil value dari input via JavaScript inline.
- **Template inline vs file** — saya pilih `fmt.Sprintf` langsung di handler untuk fragment kecil. Untuk full page saya pake `template.ParseFiles`. Lebih fleksibel karena fragment butuh atribut HTMX yang dinamis (tergantung context).
- **XSS prevention** — semua value dari job saya pass lewat `template.HTMLEscapeString()` sebelum di-embed ke HTML. Job ID kan bisa aja di-input user, jadi wajib di-escape.

## Hal yang Mungkin Bisa Di-improve

Kalau mau push lebih lanjut:

1. **Worker pool** — sekarang tiap enqueue bikin 1 goroutine. Untuk ribuan job concurrent bisa makan memory. Solusinya: buffered channel + N worker tetap.
2. **Persistence** — in-memory doang, restart = ilang. Tinggal swap `repository/inmem/` ke `repository/mysql/` atau `repository/postgres/` karena udah dipisah via interface.
3. **Metrics & tracing** — zap udah ada, tinggal tambah OpenTelemetry buat distributed tracing kalau beneran dijalanin di multiple node.
4. **Dataloader implementation** — sekarang placeholder. Isi dengan batched fetch supaya kalo ada query nested (`Jobs → Organization → User`) jadi efisien.
5. **Graceful shutdown** — kalau server di-stop, goroutine yang lagi jalan process job bakal ke-terminate paksa. Idealnya pake `context.WithCancel` + signal handler.

## Kendala Selama Pengerjaan

- **Server ngga mau mati bersih** di Windows — `Stop-Process` kadang nge-leak file handle. Solusinya tutup terminal aja, atau `taskkill /F /IM jobqueue.exe`.
- **CRLF warning di git commit** — Git di Windows suka ngingetin CRLF vs LF. Bukan error, cuma warning. Aman di-ignore.
- **GraphQL error handling** — kalau service return error, GraphQL balikin response dengan `errors: [...]` array, bukan throw. Harus dicek manual di frontend.

## Cara Jalanin

```bash
# dari folder project
go run main.go

# buka di browser
# GraphiQL:    http://localhost:58579/graphiql
# Dashboard:   http://localhost:58579/jobqueue/dashboard
```

Mutation buat test unstable-job di GraphiQL:
```graphql
mutation SimulateUnstableJob {
  Enqueue(task: "unstable-job") {
    id
    attempts
    status
  }
}
```

Tunggu sekitar 9 detik (3 attempts × 2 detik kerja + 2× 1 detik retry delay), lalu query ulang:
```graphql
query GetJobById {
  Job(id: "<id-dari-hasil-mutation>") {
    id
    status
    attempts
  }
}
```

Expected: `status: "completed"`, `attempts: 3`.

## Penutup

Overall technical test-nya seru — kombinasi GraphQL + HTMX itu cukup unik, jarang ada project yang mix dua stack itu. Paksa saya bener-bener paham bedanya server-rendered vs SPA, dan kapan polling cukup vs kapan butuh WebSocket.

Bagian tersulit buat saya: GraphQL resolver pattern. Awalnya bingung kenapa method di struct bisa otomatis ke-resolve jadi query. Setelah baca dokumentasi graph-gophers baru ngeh — konvensi nama method = nama field di schema, dan return value-nya jadi child resolver. Elegant sebenernya, tapi butuh waktu buat klik.

Bagian paling satisfying: liat unstable-job di dashboard. Klik tombol → status `pending` → 2 detik kemudian `running` → 2 detik kemudian balik `pending` (gagal attempt 1) → cycle lagi → terakhir `completed` dengan `attempts: 3`. Bener-bener keliatan retry logic-nya jalan.

Kalau ada reviewer yang baca ini — feedback-nya ditunggu. Saya tau masih banyak yang bisa di-improve terutama di bagian graceful shutdown dan observability. Untuk test ini saya fokus bikin yang functional dulu, baru polish belakangan.
