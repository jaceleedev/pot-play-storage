# API 문서

[← 이전: 아키텍처](./ARCHITECTURE.md) | [다음: 기능 명세 →](./FEATURES.md)

---

Pot Play Storage REST API의 상세 가이드입니다.

## 📖 개요

### Base URL
```
http://localhost:8090/api/v1
```

### 인증
현재 MVP 버전에서는 기본적인 API 키 인증을 사용합니다. (향후 JWT 토큰 인증 추가 예정)

### Content-Type
- **업로드**: `multipart/form-data`
- **기타 요청**: `application/json`

### 응답 형식
모든 응답은 JSON 형식이며, 표준 HTTP 상태 코드를 사용합니다.

## 📋 엔드포인트 목록

| 메소드 | 엔드포인트 | 설명 |
|--------|------------|------|
| POST | `/files` | 파일 업로드 |
| GET | `/files/{id}` | 파일 다운로드 |
| DELETE | `/files/{id}` | 파일 삭제 |
| GET | `/files` | 파일 목록 조회 |

## 📤 파일 업로드

### 요청
```http
POST /api/v1/files
Content-Type: multipart/form-data

파라미터:
- file: 업로드할 파일 (required)
```

### cURL 예시
```bash
curl -X POST http://localhost:8090/api/v1/files \
  -F "file=@example.jpg"
```

### JavaScript 예시
```javascript
const formData = new FormData();
formData.append('file', fileInput.files[0]);

fetch('http://localhost:8090/api/v1/files', {
  method: 'POST',
  body: formData
})
.then(response => response.json())
.then(data => console.log(data));
```

### 성공 응답 (201 Created)
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "example.jpg",
  "size": 1048576,
  "content_type": "image/jpeg",
  "checksum": "abc123def456789...",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 오류 응답
```json
// 400 Bad Request - 잘못된 파일
{
  "error": "file too large"
}

// 400 Bad Request - 지원하지 않는 파일 타입
{
  "error": "unsupported file type: .exe"
}

// 400 Bad Request - 빈 파일
{
  "error": "empty file not allowed"
}

// 500 Internal Server Error - 서버 오류
{
  "error": "storage service unavailable"
}
```

### 업로드 제한사항
- **최대 파일 크기**: 100MB
- **지원 파일 타입**:
  - 이미지: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.avif`
  - 동영상: `.mp4`, `.avi`, `.mov`
  - 문서: `.pdf`, `.txt`
  - 압축: `.zip`
- **파일명 길이**: 1-255자
- **위험한 파일 타입**: 실행 파일들은 업로드 불가

### 중복 파일 처리
- **자동 중복 감지**: SHA256 해시 기반으로 동일한 파일 자동 감지
- **스토리지 절약**: 동일한 파일은 한 번만 저장, 여러 참조 생성
- **투명한 처리**: API 사용자 입장에서는 일반 업로드와 동일하게 작동
- **개별 파일명**: 각 사용자는 자신만의 파일명으로 저장 가능

## 📥 파일 다운로드

### 요청
```http
GET /api/v1/files/{id}
```

### cURL 예시
```bash
# 파일을 직접 다운로드
curl http://localhost:8090/api/v1/files/123e4567-e89b-12d3-a456-426614174000 \
  -o downloaded.jpg

# 헤더만 확인
curl -I http://localhost:8090/api/v1/files/123e4567-e89b-12d3-a456-426614174000
```

### JavaScript 예시
```javascript
// 파일 다운로드 링크 생성
const downloadUrl = `http://localhost:8090/api/v1/files/${fileId}`;
window.open(downloadUrl, '_blank');

// 또는 fetch로 Blob 처리
fetch(downloadUrl)
  .then(response => response.blob())
  .then(blob => {
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'filename.jpg';
    a.click();
  });
```

### 성공 응답 (200 OK)
```http
HTTP/1.1 200 OK
Content-Type: image/jpeg
Content-Length: 1048576
Content-Disposition: attachment; filename="example.jpg"
Cache-Control: no-cache, no-store, must-revalidate
X-Content-Type-Options: nosniff

[파일 바이너리 데이터]
```

### 오류 응답
```json
// 404 Not Found - 파일이 존재하지 않음
{
  "error": "file not found"
}

// 400 Bad Request - 잘못된 ID 형식
{
  "error": "file ID required"
}
```

## 🗑️ 파일 삭제

### 요청
```http
DELETE /api/v1/files/{id}
```

### cURL 예시
```bash
curl -X DELETE http://localhost:8090/api/v1/files/123e4567-e89b-12d3-a456-426614174000
```

### JavaScript 예시
```javascript
fetch(`http://localhost:8090/api/v1/files/${fileId}`, {
  method: 'DELETE'
})
.then(response => {
  if (response.ok) {
    console.log('파일이 삭제되었습니다.');
  }
});
```

### 성공 응답 (204 No Content)
```http
HTTP/1.1 204 No Content
```

### 오류 응답
```json
// 404 Not Found - 파일이 존재하지 않음
{
  "error": "file not found"
}

// 400 Bad Request - 잘못된 ID 형식
{
  "error": "file ID required"
}

// 500 Internal Server Error - 삭제 실패
{
  "error": "failed to delete file"
}
```

## 📋 파일 목록 조회

### 요청
```http
GET /api/v1/files
```

### cURL 예시
```bash
curl http://localhost:8090/api/v1/files
```

### JavaScript 예시
```javascript
fetch('http://localhost:8090/api/v1/files')
  .then(response => response.json())
  .then(data => {
    console.log(`총 ${data.total}개의 파일`);
    data.files.forEach(file => {
      console.log(`${file.name} (${file.size} bytes)`);
    });
  });
```

### 성공 응답 (200 OK)
```json
{
  "files": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "name": "example.jpg",
      "size": 1048576,
      "content_type": "image/jpeg",
      "checksum": "abc123def456789...",
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": "789e0123-e89b-12d3-a456-426614174001",
      "name": "document.pdf",
      "size": 2097152,
      "content_type": "application/pdf",
      "checksum": "def456abc123789...",
      "created_at": "2024-01-15T09:15:00Z"
    }
  ],
  "total": 2
}
```

### 빈 목록 응답
```json
{
  "files": [],
  "total": 0
}
```

### 오류 응답
```json
// 500 Internal Server Error - 서버 오류
{
  "error": "database connection failed"
}
```

## 🚨 HTTP 상태 코드

| 코드 | 의미 | 사용 시점 |
|------|------|----------|
| 200 | OK | 파일 다운로드, 목록 조회 성공 |
| 201 | Created | 파일 업로드 성공 |
| 204 | No Content | 파일 삭제 성공 |
| 400 | Bad Request | 잘못된 요청 (파일 검증 실패 등) |
| 404 | Not Found | 존재하지 않는 파일 요청 |
| 500 | Internal Server Error | 서버 내부 오류 |

## 🔒 보안 고려사항

### 파일 업로드 보안
- **MIME 타입 검증**: 확장자와 실제 타입 일치 확인
- **파일명 검증**: 경로 탐색 공격 방지
- **크기 제한**: DoS 공격 방지
- **실행 파일 차단**: 악성 파일 업로드 방지

### 다운로드 보안
- **보안 헤더**: `X-Content-Type-Options: nosniff`
- **캐시 제어**: 민감한 파일의 캐시 방지
- **Content-Disposition**: 파일 다운로드 강제

### 일반 보안
- **입력 검증**: 모든 입력 데이터 검증
- **오류 정보 제한**: 민감한 시스템 정보 노출 방지
- **로깅**: 모든 API 호출 로그 기록

## 🧪 테스트 시나리오

### 기본 워크플로우 테스트
```bash
#!/bin/bash

# 1. 파일 업로드
echo "=== 파일 업로드 테스트 ==="
UPLOAD_RESPONSE=$(curl -s -X POST http://localhost:8090/api/v1/files \
  -F "file=@test.jpg")
echo $UPLOAD_RESPONSE

# 업로드된 파일 ID 추출
FILE_ID=$(echo $UPLOAD_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "업로드된 파일 ID: $FILE_ID"

# 2. 파일 목록 조회
echo -e "\n=== 파일 목록 조회 테스트 ==="
curl -s http://localhost:8090/api/v1/files | jq .

# 3. 파일 다운로드
echo -e "\n=== 파일 다운로드 테스트 ==="
curl -s -I http://localhost:8090/api/v1/files/$FILE_ID

# 4. 파일 삭제
echo -e "\n=== 파일 삭제 테스트 ==="
curl -s -X DELETE http://localhost:8090/api/v1/files/$FILE_ID
echo "파일 삭제 완료"

# 5. 삭제 확인
echo -e "\n=== 삭제 확인 테스트 ==="
curl -s http://localhost:8090/api/v1/files | jq .
```

### 오류 케이스 테스트
```bash
# 너무 큰 파일 업로드 (실패해야 함)
curl -X POST http://localhost:8090/api/v1/files \
  -F "file=@large_file.zip"

# 지원하지 않는 파일 타입 (실패해야 함)
curl -X POST http://localhost:8090/api/v1/files \
  -F "file=@malicious.exe"

# 존재하지 않는 파일 다운로드 (404 응답)
curl -I http://localhost:8090/api/v1/files/nonexistent-id

# 잘못된 ID 형식 (400 응답)
curl -I "http://localhost:8090/api/v1/files/invalid-uuid"
```

## 📚 SDK 및 클라이언트 라이브러리

### Python 클라이언트 예시
```python
import requests

class PotPlayBucketClient:
    def __init__(self, base_url="http://localhost:8090/api/v1"):
        self.base_url = base_url
    
    def upload_file(self, file_path):
        with open(file_path, 'rb') as f:
            files = {'file': f}
            response = requests.post(f"{self.base_url}/files", files=files)
            return response.json()
    
    def download_file(self, file_id, save_path):
        response = requests.get(f"{self.base_url}/files/{file_id}")
        with open(save_path, 'wb') as f:
            f.write(response.content)
    
    def delete_file(self, file_id):
        response = requests.delete(f"{self.base_url}/files/{file_id}")
        return response.status_code == 204
    
    def list_files(self):
        response = requests.get(f"{self.base_url}/files")
        return response.json()

# 사용 예시
client = PotPlayBucketClient()
result = client.upload_file("example.jpg")
print(f"업로드된 파일 ID: {result['id']}")
```

### Node.js 클라이언트 예시
```javascript
const axios = require('axios');
const FormData = require('form-data');
const fs = require('fs');

class PotPlayBucketClient {
  constructor(baseURL = 'http://localhost:8090/api/v1') {
    this.baseURL = baseURL;
  }

  async uploadFile(filePath) {
    const form = new FormData();
    form.append('file', fs.createReadStream(filePath));
    
    const response = await axios.post(`${this.baseURL}/files`, form, {
      headers: form.getHeaders()
    });
    return response.data;
  }

  async downloadFile(fileId, savePath) {
    const response = await axios.get(`${this.baseURL}/files/${fileId}`, {
      responseType: 'stream'
    });
    response.data.pipe(fs.createWriteStream(savePath));
  }

  async deleteFile(fileId) {
    const response = await axios.delete(`${this.baseURL}/files/${fileId}`);
    return response.status === 204;
  }

  async listFiles() {
    const response = await axios.get(`${this.baseURL}/files`);
    return response.data;
  }
}

// 사용 예시
const client = new PotPlayBucketClient();
client.uploadFile('example.jpg')
  .then(result => console.log(`업로드된 파일 ID: ${result.id}`));
```

## 🔮 향후 API 개선 계획

### Phase 1: 인증 및 권한
- JWT 토큰 기반 인증
- 사용자별 파일 격리
- API 키 관리

### Phase 2: 고급 기능
- 파일 검색 및 필터링
- 배치 작업 (다중 업로드/삭제)
- 파일 메타데이터 수정

### Phase 3: 성능 최적화
- 페이지네이션
- 부분 업로드 (resumable upload)
- CDN 통합

---

*이 API 문서는 MVP 버전을 기준으로 작성되었으며, 향후 기능 추가에 따라 업데이트될 예정입니다.*