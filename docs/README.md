# Pot Play Storage - MVP 파일 스토리지 서비스

[← 문서 목록](./index.md) | [다음: 아키텍처 →](./ARCHITECTURE.md)

---

모바일 앱을 위한 간단하고 확장 가능한 파일 스토리지 시스템의 MVP(Minimum Viable Product) 구현입니다.

## 📋 개요

Pot Play Storage는 현대적인 마이크로서비스 아키텍처를 기반으로 구축된 파일 스토리지 서비스입니다. 심플한 REST API를 통해 파일 업로드, 다운로드, 삭제, 목록 조회 기능을 제공하며, 로컬 스토리지와 SeaweedFS 분산 파일 시스템을 모두 지원합니다.

### 🎯 주요 특징

- **RESTful API**: 직관적인 REST API 설계
- **이중 스토리지 지원**: 로컬 파일시스템과 SeaweedFS 분산 스토리지
- **보안 강화**: 파일 검증, MIME 타입 검사, 경로 보안
- **고성능**: Go 기반 비동기 처리, Redis 캐싱
- **Docker 지원**: 완전한 컨테이너화 배포
- **확장 가능**: 마이크로서비스 아키텍처

### 🚀 빠른 시작

#### 1. 전체 시스템 실행
```bash
# 저장소 클론
git clone <repository-url>
cd pot-play-storage

# Docker Compose로 개발 환경 시작
docker compose -f docker-compose.dev.yml up -d

# 실행 확인
curl http://localhost:8090/api/v1/files
```

#### 2. 기본 사용법

**파일 업로드**
```bash
curl -X POST http://localhost:8090/api/v1/files \
  -F "file=@example.jpg"
```

**파일 다운로드**
```bash
curl http://localhost:8090/api/v1/files/{file-id} -o downloaded.jpg
```

**파일 목록 조회**
```bash
curl http://localhost:8090/api/v1/files
```

**파일 삭제**
```bash
curl -X DELETE http://localhost:8090/api/v1/files/{file-id}
```

## 🏗️ 프로젝트 상태

### MVP 완료 기능
- ✅ 파일 업로드/다운로드/삭제/목록
- ✅ PostgreSQL 메타데이터 저장
- ✅ Redis 캐싱
- ✅ SeaweedFS 분산 스토리지 지원
- ✅ 파일 검증 및 보안 검사
- ✅ Docker 컨테이너화

### 제한 사항 (MVP 범위)
- 📄 지원 파일 타입: 이미지(jpg, jpeg, png, gif, webp), 동영상(mp4, avi, mov), 문서(pdf, txt), 압축(zip)
- 📊 파일 크기 제한: 100MB
- 🔐 인증: 기본 API 키 (개발 중)
- 👥 사용자 관리: 미지원 (향후 추가 예정)

## 📚 문서

자세한 정보는 다음 문서를 참조하세요:

- [🏛️ 시스템 아키텍처](./ARCHITECTURE.md) - 시스템 설계 및 구성 요소
- [⚡ 기능 명세](./FEATURES.md) - 현재 기능 및 향후 로드맵
- [🔧 기술 문서](./TECHNICAL.md) - 기술 스택 및 개발 가이드
- [📖 API 문서](./API.md) - REST API 상세 가이드

## 🌟 주요 장점

### 개발자 친화적
- 단순하고 명확한 REST API
- 표준 HTTP 상태 코드 사용
- JSON 기반 응답

### 운영 효율성
- Docker Compose 원클릭 배포
- 구조화된 로깅 (Zap)
- Redis 기반 성능 최적화

### 확장성
- 마이크로서비스 아키텍처
- SeaweedFS를 통한 수평적 확장
- 스토리지 추상화로 쉬운 백엔드 변경

## 📞 지원

이 프로젝트는 MVP 단계입니다. 프로덕션 사용을 위해서는 추가 보안 및 운영 기능이 필요할 수 있습니다.

### 다음 단계
1. 인증 및 권한 관리 구현
2. 파일 버전 관리
3. 썸네일 생성
4. 배치 작업 지원
5. 모니터링 및 알림

---

*이 문서는 pot-play-storage 프로젝트의 MVP 버전을 기준으로 작성되었습니다.*