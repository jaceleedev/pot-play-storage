# Pot Play Storage

중복 제거 기능을 갖춘 고성능 파일 스토리지 시스템

## 🎯 프로젝트 현황 (2025.08.02)

### ✅ 완료된 기능
- **파일 업로드/다운로드 API** 구현 완료
- **파일 중복 제거**: SHA256 해시 기반 자동 중복 제거 (참조 카운팅)
- **SeaweedFS 통합**: 분산 파일 스토리지 연동
- **PostgreSQL**: 파일 메타데이터 및 중복 제거 정보 저장
- **Redis**: 캐싱 레이어 구현
- **Docker Compose**: 개발/프로덕션 환경 구성
- **CI/CD**: GitHub Actions 자동 배포 파이프라인
- **헬스체크**: 모든 서비스 상태 모니터링

### 🔧 최근 수정사항
- SeaweedFS Volume 서버 IP 설정 수정 (`0.0.0.0` → `seaweedfs-volume`)
- PostgreSQL 마이그레이션 스크립트 개선 (비밀번호 전달 문제 해결)
- 헬스체크 타임아웃 증가 (안정성 개선)
- Docker Compose 환경변수 처리 개선

### 🚀 프로덕션 배포
- **URL**: https://pot-storage.pot-play.com
- **API Endpoint**: `POST /api/v1/files`
- **인증**: `X-API-Key` 헤더 필요

## 📦 시스템 아키텍처

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  API Server │────▶│ PostgreSQL  │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                    ┌──────┴──────┐     ┌─────────────┐
                    │  SeaweedFS   │────▶│    Redis    │
                    └─────────────┘     └─────────────┘
```

## 🚀 빠른 시작

### 프로덕션 환경 배포

```bash
# 1. 환경 설정
cp .env.example .env
# .env 파일 편집 (DB 비밀번호, API 키 등 설정)

# 2. 배포 스크립트 실행
chmod +x deploy.sh
./deploy.sh deploy
```

### 개발 환경 실행

```bash
# 의존성 설치
go mod download

# 개발 환경 실행 (PostgreSQL, Redis, SeaweedFS 포함)
docker compose -f docker-compose.dev.yml up -d

# API 서버 실행
make run
# 또는
go run cmd/main.go
```

## 📡 API 사용법

### 파일 업로드
```bash
curl -X POST https://pot-storage.pot-play.com/api/v1/files \
  -H "X-API-Key: your-api-key" \
  -F "file=@image.jpg"
```

### 파일 조회
```bash
# 파일 보기 (브라우저에서)
https://pot-storage.pot-play.com/api/v1/files/{file-id}

# 파일 다운로드
https://pot-storage.pot-play.com/api/v1/files/{file-id}/download
```

### 파일 목록
```bash
curl https://pot-storage.pot-play.com/api/v1/files \
  -H "X-API-Key: your-api-key"
```

## 🛠️ 기술 스택

- **언어**: Go 1.22
- **프레임워크**: Gin (HTTP), pgx (PostgreSQL)
- **데이터베이스**: PostgreSQL 15, Redis 7
- **파일 스토리지**: SeaweedFS
- **컨테이너**: Docker, Docker Compose
- **CI/CD**: GitHub Actions, GitHub Container Registry
- **프록시**: Nginx (리버스 프록시)

## 📁 프로젝트 구조

```
pot-play-storage/
├── cmd/                # 애플리케이션 진입점
├── internal/           # 내부 패키지
│   ├── config/        # 설정 관리
│   ├── handler/       # HTTP 핸들러
│   ├── model/         # 데이터 모델
│   ├── repository/    # 데이터 접근 계층
│   └── service/       # 비즈니스 로직 (중복 제거 포함)
├── pkg/               # 공개 패키지
│   ├── errors/        # 에러 처리
│   ├── storage/       # 스토리지 추상화 (SeaweedFS)
│   └── validator/     # 파일 검증
├── migrations/        # DB 마이그레이션
├── .github/           # GitHub Actions 워크플로우
└── docker-compose.*   # Docker 구성 파일
```

## 🔐 보안 기능

- **API 키 인증**: 모든 API 요청에 인증 필요
- **파일 타입 검증**: 업로드 파일 MIME 타입 확인
- **파일 크기 제한**: 100MB 제한
- **SQL 인젝션 방지**: Prepared statements 사용
- **XSS 방지**: 적절한 Content-Type 헤더 설정

## 📊 주요 기능

- **파일 중복 제거**: SHA256 해시 기반 자동 중복 제거
- **참조 카운팅**: 동일 파일 다중 참조 시 스토리지 절약
- **분산 스토리지**: SeaweedFS를 통한 확장 가능한 파일 저장
- **메타데이터 캐싱**: Redis를 통한 빠른 파일 정보 접근
- **자동 정리**: 참조 카운트 0인 파일 자동 삭제

## 📚 문서

자세한 문서는 [docs/](./docs/) 디렉토리를 참조하세요:

- [시스템 아키텍처](./docs/ARCHITECTURE.md)
- [API 명세](./docs/API.md)
- [기능 명세](./docs/FEATURES.md)
- [기술 문서](./docs/TECHNICAL.md)
- [배포 가이드](./docs/DEPLOYMENT.md)

## 🐛 알려진 이슈

- ~~SeaweedFS Volume 헬스체크 실패~~ ✅ 해결됨
- ~~PostgreSQL 마이그레이션 비밀번호 인증 실패~~ ✅ 해결됨
- ~~.env 파일 인라인 주석 파싱 문제~~ ✅ 해결됨

## 🤝 기여하기

기여를 환영합니다! 이슈나 PR을 생성해주세요.

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다.