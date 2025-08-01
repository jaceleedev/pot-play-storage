# 기술 문서

[← 이전: 기능 명세](./FEATURES.md) | [문서 목록](./index.md)

---

Pot Play Storage의 기술 스택, 성능 특성, 개발 환경 설정에 대한 상세 가이드입니다.

## 🛠️ 기술 스택

### Backend Framework
- **언어**: Go 1.22+
- **웹 프레임워크**: Gin (고성능 HTTP 라우터)
- **동시성**: Goroutines + 채널 기반 비동기 처리

### 데이터 저장소
- **메타데이터 DB**: PostgreSQL 15
  - 드라이버: `pgx/v5` (고성능 네이티브 드라이버)
  - 연결 풀링: `pgxpool`
- **캐시**: Redis 7
  - 드라이버: `go-redis/v9`
  - 연결 풀링 내장

### 스토리지 시스템
- **로컬 스토리지**: 표준 파일시스템
- **분산 스토리지**: SeaweedFS
  - 고가용성, 수평적 확장
  - S3 호환 API 지원

### 주요 라이브러리

| 라이브러리 | 버전 | 용도 |
|-----------|------|------|
| `gin-gonic/gin` | v1.10.0 | HTTP 웹 프레임워크 |
| `jackc/pgx/v5` | v5.5.5 | PostgreSQL 드라이버 |
| `redis/go-redis/v9` | v9.5.1 | Redis 클라이언트 |
| `google/uuid` | v1.6.0 | UUID 생성 |
| `spf13/viper` | v1.18.2 | 설정 관리 |
| `go.uber.org/zap` | v1.27.0 | 구조화된 로깅 |
| `gabriel-vasile/mimetype` | v1.4.3 | MIME 타입 감지 |
| `golang.org/x/sync` | v0.7.0 | 동시성 유틸리티 |

## 🏗️ 프로젝트 구조

```
pot-play-storage/
├── cmd/
│   └── main.go                    # 애플리케이션 진입점
├── internal/                      # 내부 패키지 (비공개)
│   ├── config/
│   │   └── config.go             # 데이터베이스/Redis 연결 설정
│   ├── handler/
│   │   └── file.go               # HTTP 핸들러 (API 엔드포인트)
│   ├── model/
│   │   └── file.go               # 데이터 모델 정의
│   ├── repository/
│   │   └── file.go               # 데이터 액세스 계층
│   └── service/
│       └── storage.go            # 비즈니스 로직 계층
├── pkg/                          # 공개 패키지 (재사용 가능)
│   ├── errors/
│   │   └── security.go          # 오류 처리 유틸리티
│   ├── storage/
│   │   ├── interface.go         # 스토리지 인터페이스
│   │   ├── local.go             # 로컬 스토리지 구현
│   │   ├── seaweedfs.go         # SeaweedFS 구현
│   │   └── seaweedfs_simple.go  # 단순화된 SeaweedFS 구현
│   └── validator/
│       └── file.go              # 파일 검증 로직
├── configs/                      # 설정 파일
│   ├── config.yaml              # 기본 설정
│   └── config.local.yaml        # 로컬 개발 설정
├── migrations/                   # 데이터베이스 마이그레이션
│   └── 001_initial.sql
├── docs/                         # 프로젝트 문서
└── docker-compose.dev.yml        # 개발 환경 오케스트레이션
```

### 아키텍처 패턴

#### Clean Architecture (클린 아키텍처)
```
┌─────────────────┐
│    Handlers     │  ← HTTP 요청/응답 처리
├─────────────────┤
│    Services     │  ← 비즈니스 로직
├─────────────────┤
│  Repositories   │  ← 데이터 액세스
├─────────────────┤
│    Storage      │  ← 외부 스토리지 시스템
└─────────────────┘
```

#### 의존성 주입 (Dependency Injection)
```go
// 의존성 주입 예시
repo := repository.NewFileRepository(db, cache, logger)
svc := service.NewStorageService(storage, repo, logger)
handler := handler.NewFileHandler(svc, logger)
```

## ⚡ 성능 특성

### 처리량 (Throughput)
| 작업 | 단일 인스턴스 | 클러스터 (3 노드) |
|------|---------------|-------------------|
| 파일 업로드 | ~50-100 req/s | ~150-300 req/s |
| 파일 다운로드 | ~200-500 req/s | ~600-1500 req/s |
| 메타데이터 조회 | ~1000-2000 req/s | ~3000-6000 req/s |
| 파일 목록 | ~500-1000 req/s | ~1500-3000 req/s |

*성능은 하드웨어, 네트워크, 파일 크기에 따라 달라질 수 있습니다.*

### 메모리 사용량
- **기본 메모리**: ~20-50MB (실행 시)
- **파일 처리**: 스트리밍 기반으로 최소 메모리 사용
- **Redis 캐시**: 설정에 따라 조정 가능
- **연결 풀**: PostgreSQL 최대 25개, Redis 최대 10개 연결
- **해시 계산**: 파일당 일시적으로 전체 크기만큼 메모리 사용

### 중복 제거 효과
- **스토리지 절약**: 중복 파일이 많을수록 대폭 절약 (최대 90%+)
- **업로드 속도**: 중복 파일은 해시 확인만으로 즉시 완료
- **네트워크 절약**: 중복 파일 실제 전송 불필요
- **백업 효율**: 고유 파일만 백업하여 백업 시간/공간 절약

### 레이턴시 (Latency)
| 작업 | P50 | P95 | P99 |
|------|-----|-----|-----|
| 메타데이터 조회 (캐시 히트) | <5ms | <10ms | <20ms |
| 메타데이터 조회 (캐시 미스) | <50ms | <100ms | <200ms |
| 파일 업로드 시작 | <100ms | <200ms | <500ms |
| 파일 다운로드 시작 | <50ms | <100ms | <200ms |

## 🔧 확장성 고려사항

### 수평적 확장 (Horizontal Scaling)
- **API 서버**: 상태 비저장으로 설계되어 무제한 확장 가능
- **데이터베이스**: PostgreSQL 읽기 복제본 추가 가능
- **캐시**: Redis Cluster 모드 지원
- **스토리지**: SeaweedFS 볼륨 서버 동적 추가

### 수직적 확장 (Vertical Scaling)
- **CPU**: Go의 멀티코어 활용도 높음
- **메모리**: 캐시 크기 증가로 성능 향상
- **스토리지**: NVMe SSD 사용 시 I/O 성능 크게 향상

### 병목점 분석
1. **네트워크 대역폭**: 대용량 파일 전송 시 주요 제약
2. **디스크 I/O**: 로컬 스토리지 사용 시 제약
3. **데이터베이스**: 대량의 메타데이터 쿼리 시 제약
4. **캐시 적중률**: 낮을 경우 데이터베이스 부하 증가

## 🛠️ 개발 환경 설정

### 필수 요구사항
- **Go**: 1.22 이상
- **Docker**: 20.10 이상
- **Docker Compose**: 2.0 이상
- **Git**: 버전 관리

### 1. 저장소 클론 및 의존성 설치
```bash
# 저장소 클론
git clone <repository-url>
cd pot-play-storage

# Go 모듈 다운로드
go mod download

# 의존성 검증
go mod verify
```

### 2. 환경 설정
```bash
# 로컬 설정 파일 복사
cp configs/config.yaml configs/config.local.yaml

# 필요에 따라 설정 수정
vim configs/config.local.yaml
```

### 3. 개발 환경 실행
```bash
# 전체 시스템 시작 (PostgreSQL, Redis, SeaweedFS 포함)
docker compose -f docker-compose.dev.yml up -d

# 개발 서버 실행 (Go 컴파일 + 실행)
make run
# 또는
go run cmd/main.go

# 빌드만 수행
make build
```

### 4. 데이터베이스 설정
```bash
# PostgreSQL 접속 확인
docker exec -it pot-play-storage-postgres-1 psql -U bucket_user -d bucket_db

# 테이블 생성 확인
\dt

# Redis 접속 확인
docker exec -it pot-play-storage-redis-1 redis-cli
> INFO keyspace
```

### 5. SeaweedFS 관리
```bash
# 마스터 서버 상태 확인
curl http://localhost:9333/cluster/status

# 볼륨 정보 확인
curl http://localhost:9333/dir/status

# 파일러 상태 확인
curl http://localhost:8888/dir/status
```

## 🧪 테스트 및 품질 관리

### 테스트 실행
```bash
# 전체 테스트 실행
go test ./...

# 커버리지 포함 테스트
go test -cover ./...

# 벤치마크 테스트
go test -bench=. ./...

# 테스트 상세 출력
go test -v ./...
```

### 코드 품질 도구
```bash
# 코드 포맷팅
go fmt ./...

# 정적 분석
go vet ./...

# 모듈 정리
go mod tidy

# 취약점 검사 (golang.org/x/vuln/cmd/govulncheck)
govulncheck ./...
```

### 성능 프로파일링
```bash
# CPU 프로파일링
go test -cpuprofile cpu.prof -bench .

# 메모리 프로파일링
go test -memprofile mem.prof -bench .

# 프로파일 분석
go tool pprof cpu.prof
```

## 🐳 Docker Compose 파일 가이드

Pot Play Storage 프로젝트는 개발과 프로덕션 환경을 위한 별도의 Docker Compose 설정을 제공합니다.

### 📁 파일 구조

```
pot-play-storage/
├── docker-compose.dev.yml      # 개발 환경 (로컬 개발용)
└── deploy/
    └── docker-compose.prod.yml # 프로덕션 환경 (배포용)
```

### 🔧 개발 환경 - `docker-compose.dev.yml`

**용도**: 로컬 개발 및 테스트

**특징**:
- 모든 포트가 localhost에 노출되어 디버깅 용이
- 개발용 하드코딩된 설정 (bucket_user/bucket_password)
- 볼륨 마운트로 코드 변경사항 즉시 반영
- 단순한 설정으로 빠른 시작 가능

**사용법**:
```bash
# 개발 환경 시작
docker compose -f docker-compose.dev.yml up -d

# 로그 확인
docker compose -f docker-compose.dev.yml logs -f

# 서비스 중지
docker compose -f docker-compose.dev.yml down
```

**노출 포트**:
- API: 8090
- PostgreSQL: 5432
- Redis: 6379
- SeaweedFS Master: 9333
- SeaweedFS Volume: 8081
- SeaweedFS Filer: 8888
- SeaweedFS WebDAV: 7333

### 🚀 프로덕션 환경 - `deploy/docker-compose.prod.yml`

**용도**: 프로덕션 배포

**특징**:
- 127.0.0.1 바인딩으로 보안 강화
- 환경 변수를 통한 설정 관리
- 헬스체크 및 재시작 정책 포함
- 로깅 및 모니터링 설정
- 보안 옵션 적용 (no-new-privileges)

**사용법**:
```bash
# 환경 변수 파일 준비
cp deploy/.env.example .env
# .env 파일 편집 필요

# 프로덕션 환경 시작
docker compose -f deploy/docker-compose.prod.yml up -d

# 상태 확인
docker compose -f deploy/docker-compose.prod.yml ps

# 로그 확인
docker compose -f deploy/docker-compose.prod.yml logs -f api
```

**보안 특징**:
- 포트는 127.0.0.1에만 바인딩
- 보안 헤더 및 설정 적용
- 환경 변수를 통한 크리덴셜 관리
- 컨테이너 보안 옵션 활성화

### 🔄 마이그레이션 가이드

기존 `docker-compose.yml`을 사용하던 경우:

```bash
# 기존 방식 (더 이상 작동하지 않음)
docker compose up -d

# 새로운 방식 (개발 환경)
docker compose -f docker-compose.dev.yml up -d

# 새로운 방식 (프로덕션 환경)
docker compose -f deploy/docker-compose.prod.yml up -d
```

### 📊 환경별 비교

| 항목 | 개발 환경 | 프로덕션 환경 |
|------|-----------|---------------|
| **설정 관리** | 하드코딩 | 환경 변수 |
| **포트 노출** | 모든 포트 | 필요한 포트만 |
| **보안** | 기본 설정 | 강화된 보안 |
| **헬스체크** | 없음 | 포함 |
| **로깅** | 기본 | 구조화됨 |
| **재시작 정책** | 없음 | unless-stopped |
| **볼륨 바인딩** | 개발용 | 프로덕션용 |

### 💡 사용 권장사항

#### 개발자
- 로컬 개발: `docker-compose.dev.yml` 사용
- 빠른 테스트 및 디버깅에 최적화

#### DevOps/운영자
- 프로덕션 배포: `deploy/docker-compose.prod.yml` 사용
- [배포 가이드](./DEPLOYMENT.md) 참조

#### CI/CD
- 테스트 환경: `docker-compose.dev.yml` 
- 프로덕션 배포: `deploy/docker-compose.prod.yml`

### 개발용 컨테이너 빌드

```bash
# 개발 이미지 빌드
docker build -t pot-play-storage:dev .

# 개발 컨테이너 실행
docker run -p 8090:8090 pot-play-storage:dev
```

### 볼륨 매핑

```yaml
volumes:
  - ./uploads:/app/uploads      # 로컬 파일 저장
  - ./configs:/app/configs      # 설정 파일
  - postgres_data:/var/lib/postgresql/data  # DB 데이터
```

## 🔧 설정 관리

### 설정 파일 구조
```yaml
server:
  port: 8090                    # API 서버 포트
  read_timeout: 30s            # 읽기 타임아웃
  write_timeout: 30s           # 쓰기 타임아웃

database:
  host: postgres               # DB 호스트
  port: 5432                   # DB 포트
  user: bucket_user            # DB 사용자
  password: bucket_password    # DB 패스워드
  name: bucket_db              # DB 이름
  ssl_mode: disable            # SSL 모드

redis:
  host: redis                  # Redis 호스트
  port: 6379                   # Redis 포트
  db: 0                        # Redis DB 번호

storage:
  type: seaweedfs              # 스토리지 타입 (local|seaweedfs)
  local_path: ./uploads        # 로컬 스토리지 경로
  seaweedfs:
    master_url: seaweedfs-master:9333  # SeaweedFS 마스터 URL

security:
  api_key: your-api-key        # API 키 (기본 인증)

log:
  level: info                  # 로그 레벨 (debug|info|warn|error)
```

### 환경별 설정
- **개발환경**: `config.local.yaml` (Git 무시)
- **테스트환경**: `config.test.yaml`
- **프로덕션환경**: 환경변수 또는 외부 설정 관리

## 📊 모니터링 및 로깅

### 구조화된 로깅 (Zap)
```go
logger.Info("file uploaded successfully",
    zap.String("file_id", fileID),
    zap.String("filename", filename),
    zap.Int64("size", fileSize),
    zap.Duration("duration", processingTime),
)
```

### 로그 레벨
- **DEBUG**: 개발 시 상세 정보
- **INFO**: 일반적인 작업 정보
- **WARN**: 주의가 필요한 상황
- **ERROR**: 오류 상황

### 메트릭 수집 (향후 계획)
- **HTTP 메트릭**: 응답 시간, 상태 코드, 처리량
- **스토리지 메트릭**: 업로드/다운로드 속도, 용량
- **데이터베이스 메트릭**: 쿼리 시간, 연결 풀 상태
- **시스템 메트릭**: CPU, 메모리, 디스크 사용량

## 📝 핵심 코드 구현

### Storage 인터페이스 (pkg/storage/interface.go)

```go
package storage

import (
	"context"
	"io"
)

type Storage interface {
	Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error)
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, prefix string) ([]string, error)
}
```

### Local Storage 구현 (pkg/storage/local.go)

```go
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

func (s *LocalStorage) Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error) {
	fullPath := filepath.Join(s.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return io.Copy(file, reader)
}

func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.basePath, path))
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	return os.Remove(filepath.Join(s.basePath, path))
}

func (s *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	searchPath := filepath.Join(s.basePath, prefix)
	err := filepath.Walk(searchPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(s.basePath, p)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}
```

### 파일 검증 로직 (pkg/validator/file.go)

```go
package validator

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

var AllowedTypes = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".mp4": true, ".avi": true, ".mov": true,
}

func ValidateFile(header *multipart.FileHeader) error {
	if header.Size > 1<<30 { // 1GB
		return fmt.Errorf("file too large")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !AllowedTypes[ext] {
		return fmt.Errorf("unsupported type: %s", ext)
	}
	return nil
}
```

### 파일 모델 (internal/model/file.go)

```go
package model

import "time"

type File struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	Checksum    string    `json:"checksum"`
	CreatedAt   time.Time `json:"created_at"`
}
```

## ⚙️ SeaweedFS 통합 가이드

### SeaweedFS 개요

SeaweedFS는 분산 파일 시스템으로 대용량 파일 처리와 수평적 확장을 지원합니다.

### 구성 요소

1. **Master Server** (Port 9333)
   - 메타데이터 및 파일 ID 할당 관리
   - 볼륨 서버 조정
   - 클러스터 상태 및 헬스 체크 제공

2. **Volume Server** (Port 8081)
   - 실제 파일 데이터 저장
   - 파일 업로드 및 다운로드 처리
   - 스토리지 볼륨 관리

3. **Filer** (Port 8888)
   - 파일시스템 인터페이스 제공
   - 디렉토리 작업 처리
   - 파일 경로를 볼륨 위치에 매핑

### 설정 방법

#### 1. SeaweedFS 서비스 시작

```bash
# Docker Compose로 SeaweedFS 시작
docker-compose up -d seaweedfs-master seaweedfs-volume seaweedfs-filer

# 또는 개발 환경 docker-compose 사용
# configs/config.yaml에서 storage.type: seaweedfs 설정
```

#### 2. 설정 파일 업데이트

```yaml
storage:
  type: seaweedfs
  seaweedfs:
    master_url: seaweedfs-master:9333  # 또는 localhost:9333 (로컬 개발용)
```

#### 3. 통합 테스트

```bash
# 파일 업로드
curl -X POST -F "file=@test.txt" http://localhost:8090/api/v1/files

# 파일 목록
curl http://localhost:8090/api/v1/files

# 파일 다운로드 (업로드 응답에서 받은 ID 사용)
curl http://localhost:8090/api/v1/files/{file-id}

# 파일 삭제
curl -X DELETE http://localhost:8090/api/v1/files/{file-id}
```

### SeaweedFS 관리 인터페이스

- **Master Web UI**: http://localhost:9333
- **Filer Interface**: http://localhost:8888
- **Volume Server**: http://localhost:8081

### 구현 방식

#### Simple Implementation (권장 - 개발용)
- SeaweedFS Filer HTTP API 직접 사용
- 간단한 설정 및 디버깅
- 단일 노드 배포에 적합

#### Full Implementation (프로덕션용)
- Master 서버를 통한 파일 ID 할당
- 분산 볼륨 관리 지원
- 복잡하지만 더 나은 확장성 제공

### 문제 해결

#### 일반적인 문제

1. **Connection Refused**
   ```bash
   # 서비스 실행 상태 확인
   docker-compose ps
   
   # 로그 확인
   docker-compose logs seaweedfs-master
   ```

2. **업로드 실패**
   ```bash
   # Filer 접근성 확인
   curl http://localhost:8888/
   
   # Volume 서버 상태 확인
   curl http://localhost:9333/vol/status
   ```

3. **File Not Found**
   ```bash
   # Filer를 통한 파일 목록 확인
   curl http://localhost:8888/?pretty=y
   
   # Master에서 볼륨 할당 확인
   curl http://localhost:9333/dir/status
   ```

#### 헬스 체크

```bash
# Master 서버 헬스
curl http://localhost:9333/cluster/status

# Volume 서버 헬스
curl http://localhost:8081/status

# Filer 헬스
curl http://localhost:8888/
```

### 성능 튜닝

#### SeaweedFS 설정

```bash
# 더 많은 스토리지를 가진 Volume 서버
docker run chrislusf/seaweedfs volume \
  -mserver=master:9333 \
  -dir=/data1,/data2,/data3 \
  -max=1000

# 커스텀 설정을 가진 Master
docker run chrislusf/seaweedfs master \
  -port=9333 \
  -mdir=/data \
  -defaultReplication=001
```

#### 애플리케이션 설정

```yaml
storage:
  seaweedfs:
    master_url: seaweedfs-master:9333
    timeout: 30s
    max_idle_conns: 100
    max_conns_per_host: 10
```

### 개발 vs 프로덕션

#### 개발 설정
- 단일 Master, 단일 Volume 서버, 단일 Filer
- 복제 없음
- 테스트 및 개발에 적합
- `SeaweedFSSimpleStorage` 구현 사용

#### 프로덕션 고려사항
- 중복성을 위한 다중 Volume 서버
- Master 서버 복제
- 적절한 백업 전략
- 모니터링 및 알림
- 전체 `SeaweedFSStorage` 구현 사용

## 🔧 전체 의존성 명세 (go.mod)

```go
module pot-play-storage

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/redis/go-redis/v9 v9.5.1
	github.com/spf13/viper v1.18.2
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.7.0
)

require (
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
```

## 🐳 Docker 설정

### Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY configs /app/configs
VOLUME /app/uploads
EXPOSE 8090
CMD ["./server"]
```

### Docker Compose 설정

```yaml
version: "3.8"
services:
  api:
    build: .
    ports:
      - "8090:8090"
    volumes:
      - ./uploads:/app/uploads
    depends_on:
      - postgres
      - redis
      - seaweedfs-master
      - seaweedfs-volume
      - seaweedfs-filer

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: bucket_user
      POSTGRES_PASSWORD: bucket_password
      POSTGRES_DB: bucket_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  seaweedfs-master:
    image: chrislusf/seaweedfs:latest
    command: master -ip=seaweedfs-master -port=9333 -mdir=/data
    volumes:
      - seaweedfs_master_data:/data
    ports:
      - "9333:9333"

  seaweedfs-volume:
    image: chrislusf/seaweedfs:latest
    command: volume -ip=seaweedfs-volume -port=8090 -dir=/data -mserver=seaweedfs-master:9333
    volumes:
      - seaweedfs_volume_data:/data
    ports:
      - "8081:8090"
    depends_on:
      - seaweedfs-master

  seaweedfs-filer:
    image: chrislusf/seaweedfs:latest
    command: filer -ip=seaweedfs-filer -port=8888 -master=seaweedfs-master:9333
    volumes:
      - seaweedfs_filer_data:/data
    ports:
      - "8888:8888"
    depends_on:
      - seaweedfs-master
      - seaweedfs-volume

volumes:
  postgres_data:
  seaweedfs_master_data:
  seaweedfs_volume_data:
  seaweedfs_filer_data:
```

### Makefile

```makefile
.PHONY: build run test docker-up docker-down

build:
	go build -o bin/server ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
```

## 📝 최근 변경사항

### 파일 및 설정 업데이트
- **Docker 최적화**: `.dockerignore`에 `configs/config.local.yaml` 항목 추가
- **데이터베이스 스키마**: 마이그레이션 `003_make_legacy_columns_nullable.sql` 추가
  - 레거시 컬럼들을 nullable로 변경하여 새로운 deduplication 스키마와 호환성 개선

### SeaweedFS 통합 개선
- **Filer URL 생성 수정**: SeaweedFS 클라이언트에서 올바른 Filer URL 생성 로직 구현
- **연결 안정성 향상**: SeaweedFS 마스터 서버와의 연결 관리 개선

### 파일 검증 강화
- **MIME 타입 검증 개선**: 텍스트 파일의 charset 포함 MIME 타입 정확한 검증
  - `text/plain; charset=utf-8` 형태의 MIME 타입 지원
- **확장자-MIME 타입 매칭**: 파일 확장자와 MIME 타입 간의 정확한 매칭 로직 구현

### 보안 및 안정성
- **파일 업로드 검증**: 파일 타입 및 크기 제한 강화
- **에러 처리**: 보안을 고려한 에러 메시지 sanitization 적용

---

*이 문서는 개발팀을 위한 기술 참조 가이드입니다. 프로덕션 배포 시 추가적인 보안 및 성능 설정이 필요할 수 있습니다.*