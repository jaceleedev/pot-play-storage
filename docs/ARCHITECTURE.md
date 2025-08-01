# 시스템 아키텍처

[← 이전: README](./README.md) | [다음: API →](./API.md)

---

Pot Play Storage의 시스템 아키텍처와 구성 요소를 설명합니다.

## 🏗️ 전체 아키텍처

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client Apps   │────│   API Gateway    │────│   Load Balancer │
│  (Mobile/Web)   │    │   (Future)       │    │    (Future)     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │     API Server          │
                    │   (Go + Gin)            │
                    │   Port: 8090            │
                    └─────────────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   PostgreSQL    │    │     Redis       │    │    Storage      │
│  (Metadata)     │    │   (Cache)       │    │   (Files)       │
│  Port: 5432     │    │  Port: 6379     │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                       ┌────────────────┴────────────────┐
                                       │                                 │
                              ┌─────────────────┐              ┌─────────────────┐
                              │ Local Storage   │              │   SeaweedFS     │
                              │ (Development)   │              │ (Production)    │
                              └─────────────────┘              └─────────────────┘
                                                                        │
                                                          ┌─────────────┼─────────────┐
                                                          │             │             │
                                                   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
                                                   │   Master    │ │   Volume    │ │   Filer     │
                                                   │ Port: 9333  │ │ Port: 8081  │ │ Port: 8888  │
                                                   └─────────────┘ └─────────────┘ └─────────────┘
```

## 🔧 핵심 구성요소

### 1. API Server (Go + Gin)
**역할**: RESTful API 제공 및 비즈니스 로직 처리

**구성 요소**:
- **Handler Layer**: HTTP 요청/응답 처리
- **Service Layer**: 비즈니스 로직 구현
- **Repository Layer**: 데이터 액세스 추상화

**특징**:
- Graceful shutdown 지원
- 구조화된 로깅 (Zap)
- 미들웨어 기반 확장성

### 2. PostgreSQL (메타데이터 저장소)
**역할**: 파일 메타데이터 및 시스템 데이터 저장

**스키마**:
```sql
-- 고유 파일 콘텐츠 저장
CREATE TABLE storage (
    hash VARCHAR(64) PRIMARY KEY,
    size BIGINT NOT NULL,
    content_type VARCHAR(255),
    storage_path VARCHAR(500),
    reference_count INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 사용자별 파일 참조
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    storage_hash VARCHAR(64) REFERENCES storage(hash),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- 기존 데이터 호환성을 위한 레거시 컬럼
    size BIGINT,
    content_type VARCHAR(100),
    storage_path VARCHAR(500),
    checksum VARCHAR(64)
);
```

**인덱스**:
- `idx_storage_hash`: 빠른 해시 조회
- `idx_files_storage_hash`: 파일-스토리지 연결
- `idx_files_created_at`: 시간순 정렬용
- `idx_files_checksum`: 레거시 호환성

### 3. Redis (캐시 시스템)
**역할**: 성능 최적화를 위한 메모리 캐시

**캐시 전략**:
- **파일 메타데이터**: 1시간 TTL
- **파일 목록**: 10분 TTL
- **Write-through**: 업데이트 시 캐시 무효화

### 4. Storage Abstraction (스토리지 추상화)
**역할**: 다양한 스토리지 백엔드 지원

**인터페이스**:
```go
type Storage interface {
    Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error)
    Get(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    List(ctx context.Context, prefix string) ([]string, error)
}
```

**구현체**:
- **LocalStorage**: 로컬 파일시스템 기반 (개발/테스트용)
- **SeaweedFSStorage**: 분산 파일시스템 (프로덕션용)

## 📊 데이터 플로우

### 파일 업로드 플로우
```
1. Client → API Server: POST /api/v1/files (multipart/form-data)
2. API Server: 파일 검증 (크기, 타입, 보안)
3. API Server: SHA256 해시 계산
4. API Server → PostgreSQL: 해시로 기존 파일 확인
5. (중복인 경우) 
   a. API Server → PostgreSQL: reference_count 증가
   b. API Server → PostgreSQL: files 테이블에 참조 추가
6. (신규 파일인 경우)
   a. API Server → Storage: 파일 저장
   b. API Server → PostgreSQL: storage 테이블 추가
   c. API Server → PostgreSQL: files 테이블에 참조 추가
7. API Server → Redis: 캐시 갱신
8. API Server → Client: 파일 정보 응답
```

### 파일 다운로드 플로우
```
1. Client → API Server: GET /api/v1/files/{id}
2. API Server → Redis: 메타데이터 캐시 확인
3. (캐시 미스) API Server → PostgreSQL: 메타데이터 조회
4. API Server → Storage: 파일 스트림 조회
5. API Server → Client: 파일 스트림 전송
```

### 파일 삭제 플로우
```
1. Client → API Server: DELETE /api/v1/files/{id}
2. API Server → PostgreSQL: 파일 정보 조회
3. API Server → PostgreSQL: files 테이블에서 참조 삭제
4. API Server → PostgreSQL: reference_count 감소
5. (reference_count = 0인 경우)
   a. API Server → Storage: 실제 파일 삭제
   b. API Server → PostgreSQL: storage 테이블에서 삭제
6. API Server → Redis: 캐시 무효화
7. API Server → Client: 성공 응답
```

## 🔒 보안 아키텍처

### 파일 검증 레이어
1. **파일 크기 검증**: 100MB 제한
2. **MIME 타입 검증**: 허용된 타입만 업로드
3. **파일명 검증**: 경로 탐색 공격 방지
4. **확장자 검증**: 위험한 실행 파일 차단

### 보안 헤더
```go
c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
c.Header("X-Content-Type-Options", "nosniff")
c.Header("Content-Disposition", "attachment; filename=\"...\"")
```

## 🚀 확장성 설계

### 수평적 확장 가능 요소
- **API Server**: 로드밸런서를 통한 다중 인스턴스
- **SeaweedFS**: 분산 파일시스템으로 스토리지 확장
- **Redis**: Redis Cluster로 캐시 확장
- **PostgreSQL**: 읽기 복제본 추가 가능

### 수직적 확장 가능 요소
- **메모리**: Redis 캐시 크기 증가
- **CPU**: Go의 멀티코어 활용
- **스토리지**: SeaweedFS 볼륨 서버 추가

## 🐳 배포 아키텍처

### Development (Docker Compose)
```yaml
services:
  api:          # API Server
  postgres:     # 메타데이터 DB
  redis:        # 캐시
  seaweedfs-*:  # 분산 스토리지 (Master, Volume, Filer)
```

### Production (권장 구성)
```
┌─────────────────┐
│ Load Balancer   │ (nginx/HAProxy)
├─────────────────┤
│ API Servers     │ (multiple instances)
├─────────────────┤
│ Database        │ (PostgreSQL cluster)
├─────────────────┤
│ Cache           │ (Redis cluster)
├─────────────────┤
│ Storage         │ (SeaweedFS cluster)
└─────────────────┘
```

## 📈 성능 특성

### 처리량 (단일 인스턴스 기준)
- **업로드**: ~100MB/s (네트워크 대역폭 의존)
- **다운로드**: ~200MB/s (캐시 및 스토리지 성능 의존)
- **메타데이터 조회**: ~1000 req/s (Redis 캐시 활용 시)

### 레이턴시
- **파일 업로드**: 파일 크기에 비례
- **메타데이터 조회**: <10ms (캐시 히트 시)
- **파일 다운로드 시작**: <50ms

## 🔄 설계 원칙

### 1. 단일 책임 원칙
각 레이어와 컴포넌트는 명확한 단일 책임을 가집니다.

### 2. 의존성 역전
인터페이스를 통한 추상화로 구현체 변경이 용이합니다.

### 3. 확장 가능성
새로운 스토리지 백엔드나 기능 추가가 용이한 구조입니다.

### 4. 장애 격리
각 컴포넌트의 장애가 전체 시스템에 미치는 영향을 최소화합니다.

---

*이 아키텍처는 MVP 단계를 기준으로 하며, 프로덕션 환경에서는 추가적인 보안, 모니터링, 백업 등이 필요할 수 있습니다.*