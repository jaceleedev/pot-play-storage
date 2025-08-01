# Pot Play Storage

중복 제거 기능을 갖춘 고성능 파일 스토리지 시스템

## 🚀 빠른 시작

### 개발 환경 실행

```bash
# 의존성 설치
go mod download

# 환경 설정 (프로덕션용, 개발은 기본 설정 사용)
cp .env.example .env
# .env 파일을 열고 필요한 값 수정 (선택사항)

# 개발 환경 실행 (PostgreSQL, Redis, SeaweedFS 포함)
docker compose -f docker-compose.dev.yml up -d

# API 서버 실행
make run
# 또는
go run cmd/main.go
```

### API 테스트

```bash
# 파일 업로드
curl -X POST -F "file=@test.txt" http://localhost:8090/api/v1/files

# 파일 목록 조회
curl http://localhost:8090/api/v1/files

# 파일 다운로드
curl http://localhost:8090/api/v1/files/{file-id}
```

## 📚 문서

자세한 문서는 [docs/](./docs/) 디렉토리를 참조하세요:

- [시스템 아키텍처](./docs/ARCHITECTURE.md)
- [API 명세](./docs/API.md)
- [기능 명세](./docs/FEATURES.md)
- [기술 문서](./docs/TECHNICAL.md)
- [배포 가이드](./docs/DEPLOYMENT.md)

## 🛠️ 프로젝트 구조

```
pot-play-storage/
├── cmd/                # 애플리케이션 진입점
├── internal/           # 내부 패키지
│   ├── config/        # 설정 관리
│   ├── handler/       # HTTP 핸들러
│   ├── model/         # 데이터 모델
│   ├── repository/    # 데이터 접근 계층
│   └── service/       # 비즈니스 로직
├── pkg/               # 공개 패키지
│   ├── storage/       # 스토리지 추상화
│   └── validator/     # 파일 검증
├── configs/           # 설정 파일
├── migrations/        # DB 마이그레이션
├── deploy/            # 배포 관련 파일
└── tests/             # 테스트 스크립트

```

## 🔧 주요 기능

- **파일 중복 제거**: SHA256 해시 기반 자동 중복 제거
- **다중 스토리지 지원**: 로컬 파일시스템, SeaweedFS
- **고성능 캐싱**: Redis 기반 메타데이터 캐싱
- **실시간 기능**: WebSocket 지원
- **보안**: 파일 타입 검증, 크기 제한

## 🤝 기여하기

기여를 환영합니다! 이슈나 PR을 생성해주세요.

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다.