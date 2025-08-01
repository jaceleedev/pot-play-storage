# 📚 Pot Play Storage 문서 가이드

## 문서 읽기 순서

이 가이드는 Pot Play Storage 프로젝트를 효과적으로 이해하기 위한 문서 읽기 순서를 제공합니다.

### 🎯 추천 읽기 순서

#### 1️⃣ **시작하기** - [README.md](./README.md)
- 프로젝트 개요 및 목적
- 빠른 시작 가이드
- 기본 사용법
- MVP 상태 설명

#### 2️⃣ **아키텍처 이해** - [ARCHITECTURE.md](./ARCHITECTURE.md)
- 시스템 구성 요소
- 데이터 플로우
- 설계 원칙
- 확장성 고려사항

#### 3️⃣ **API 사용법** - [API.md](./API.md)
- REST API 엔드포인트
- 요청/응답 형식
- 에러 처리
- 실제 사용 예시

#### 4️⃣ **기능 명세** - [FEATURES.md](./FEATURES.md)
- 현재 구현된 기능
- 향후 개발 로드맵
- 기능별 상세 설명

#### 5️⃣ **기술 심화** - [TECHNICAL.md](./TECHNICAL.md)
- 기술 스택 상세
- 코드 구조
- SeaweedFS 설정
- 성능 최적화

#### 6️⃣ **배포 가이드** - [DEPLOYMENT.md](./DEPLOYMENT.md)
- 프로덕션 배포 방법
- 서버 설정 및 환경 구성
- 모니터링 및 유지보수
- 문제 해결 가이드

#### 7️⃣ **CentOS 8 가이드** - [CENTOS8.md](./CENTOS8.md)
- CentOS 8 특수 설정
- Ubuntu와의 차이점
- SELinux 및 firewalld 설정
- 문제 해결 방법

---

### 👥 대상별 읽기 가이드

#### **처음 사용자**
1. README.md → API.md

#### **개발자**
1. README.md → ARCHITECTURE.md → TECHNICAL.md

#### **운영자/DevOps**
1. README.md → TECHNICAL.md (Docker/SeaweedFS 섹션) → DEPLOYMENT.md
2. CentOS 8 사용 시: CENTOS8.md 추가 필독

#### **프로젝트 기여자**
1. 전체 문서 순서대로 읽기

---

### 📖 문서 구조

```
docs/
├── index.md          # 이 파일 (문서 가이드)
├── README.md         # 프로젝트 개요
├── ARCHITECTURE.md   # 시스템 아키텍처
├── API.md           # API 문서
├── FEATURES.md      # 기능 명세
├── TECHNICAL.md     # 기술 상세
├── DEPLOYMENT.md    # 배포 가이드
└── CENTOS8.md       # CentOS 8 특수 설정
```

### 💡 팁

- 각 문서 상단에는 다음/이전 문서로의 네비게이션이 있습니다
- 코드 예시는 복사하여 바로 실행할 수 있도록 작성되었습니다
- 문서에 문제가 있거나 개선사항이 있다면 이슈를 등록해주세요