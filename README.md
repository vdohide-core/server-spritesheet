# server-spritesheet

Dedicated worker สำหรับสร้าง sprite sheet + `sprite.vtt` แยกจาก `server-transcode`

## Pipeline

```
server-transcode → video renditions (1080/720/480/360)
server-spritesheet → sprite/sprite-1.jpg + sprite.vtt
server-transfer → ดึง sprite.zip จาก S3 tmp (กรณี remote)
server-static / server-player → serve sprite URLs
```

## การทำงาน

Worker จะ claim ไฟล์ `ready` ที่มี video media (original หรือ transcoded) แต่ยังไม่มี `medias.type = thumbnail`

**ไม่ต้องรอ server-transcode** — ใช้ `file_original.mp4` ได้เลยหลัง server-transfer

### 1. เตรียม input video

| เงื่อนไข | แหล่ง input |
|----------|-------------|
| `STORAGE_ID` ตรงกับ storage ของ video และไฟล์อยู่บน disk | `/home/files/{fileId}/{fileName}` โดยตรง |
| อยู่คนละที่ | ดาวน์โหลด `http://{storageHost}:8888/{mediaSlug}.mp4` |

เลือก video ขนาดเล็กสุด: 360 → 480 → 720 → 1080 → **original**

### 2. สร้าง sprite

- Grid **6×6**
- Interval **1000ms** (1 วินาทีต่อเฟรม)
- Output: `sprite/sprite-1.jpg`, `sprite-2.jpg`, …, `sprite.vtt`

### 3. ติดตั้งผลลัพธ์

| เงื่อนไข | การติดตั้ง |
|----------|------------|
| Co-located (`STORAGE_ID` = storage ของ video) | ย้ายไป `/home/files/{fileId}/sprite/*` + สร้าง `medias` thumbnail + `cloneFrom` |
| Remote | zip → อัปโหลด S3 `{fileId}/sprite.zip` (ให้ `server-transfer` ติดตั้งต่อ) |

## Config (.env)

```env
MONGODB_URI=mongodb+srv://...
STORAGE_ID=          # UUID ของ storage เมื่อรันบนเครื่อง storage
STORAGE_PATH=/home/files
PORT=8084
WORKER_ID=hostname@1  # ตั้งโดย systemd
```

## MongoDB settings

```js
db.settings.updateOne(
  { name: "spritesheet_enabled" },
  { $set: { name: "spritesheet_enabled", value: true } },
  { upsert: true }
)
```

## ติดตั้ง (Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/vdohide-core/server-spritesheet/main/install.sh | sudo -E bash -s -- \
  --mongodb-uri "mongodb+srv://..." \
  --storage-id "YOUR_STORAGE_UUID" \
  --storage-path /home/files \
  --count 1
```

## Build

### Windows

```bat
build.bat
```

### Linux / dev

```bash
go build -o server-spritesheet ./cmd
```

## Log viewer

```
http://localhost:8084/ui
```

## โครงสร้างบน storage

```
/home/files/{fileId}/
  file_360.mp4
  file_720.mp4
  ...
  sprite/
    sprite-1.jpg
    sprite-2.jpg
    sprite.vtt
```

## Media record

```json
{
  "type": "thumbnail",
  "fileName": "sprite.vtt",
  "storageId": "...",
  "fileId": "..."
}
```

## อ้างอิง

- **server-transcode** — sprite generation (`thumbnail.go`), `cloneMediaToClonedFiles`, thumbnail media
- **server-download** — S3 temp upload (`s3upload.go`)

ไม่ใช้ SCP/Node scripts — remote input ผ่าน HTTP, remote output ผ่าน S3 zip

## หมายเหตุ

- ปิดการสร้าง sprite ใน `server-transcode` (STEP 5) เมื่อใช้ worker นี้แล้ว เพื่อไม่ให้ซ้ำซ้อน
- Remote path ใช้ S3 storage ที่ `accepts` มี `temp` + `video` (เหมือน `server-download`)
- `processType`: `spritesheet` ใน collection `video_process`
