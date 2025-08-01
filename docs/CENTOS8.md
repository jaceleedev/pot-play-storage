# CentOS 8 전용 설정 가이드

CentOS 8 서버에서 Pot Play Storage를 설정하는 방법입니다.

## 주요 차이점 (Ubuntu/Debian과 비교)

### 패키지 관리자
- Ubuntu: `apt`, `apt-get`
- CentOS 8: `dnf` (이전 `yum`의 후속)

### 방화벽
- Ubuntu: `ufw`
- CentOS 8: `firewalld`

### 파일 경로
- Nginx 설정:
  - Ubuntu: `/etc/nginx/sites-available/`, `/etc/nginx/sites-enabled/`
  - CentOS 8: `/etc/nginx/conf.d/`

### 시스템 서비스
- 모두 `systemctl` 사용 (동일)
- CentOS 8은 SELinux 추가 보안 계층 존재

## CentOS 8 특수 설정

### SELinux 관리

```bash
# SELinux 상태 확인
getenforce

# Docker와 충돌 시 Permissive 모드로 설정
sudo setenforce 0

# 영구 설정 (재부팅 후에도 유지)
sudo vi /etc/selinux/config
# SELINUX=permissive 로 변경
```

### Firewalld 추가 설정

```bash
# Docker 네트워크를 위한 마스커레이딩 활성화
sudo firewall-cmd --permanent --zone=public --add-masquerade

# Docker 인터페이스 신뢰
sudo firewall-cmd --permanent --zone=trusted --add-interface=docker0

# 변경사항 적용
sudo firewall-cmd --reload
```

### EPEL 저장소

CentOS 8에서 추가 패키지를 위해 EPEL(Extra Packages for Enterprise Linux) 저장소가 필요합니다:

```bash
sudo dnf install -y epel-release
```

## Docker Compose 명령어 차이

CentOS 8에서 Docker Compose Plugin 설치 시:
- 기존: `docker-compose` (독립 실행 파일)
- 신규: `docker compose` (Docker 플러그인)

```bash
# 기존 방식
docker-compose -f docker-compose.yml up -d

# 새로운 방식 (권장)
docker compose -f docker-compose.yml up -d
```

## 문제 해결

### 1. SELinux 관련 문제
```bash
# Docker 컨테이너가 파일에 접근할 수 없는 경우
sudo chcon -Rt svirt_sandbox_file_t /opt/pot-storage/data
```

### 2. Firewalld와 Docker 충돌
```bash
# Docker가 iptables 규칙과 충돌하는 경우
sudo firewall-cmd --permanent --direct --add-rule ipv4 filter DOCKER-USER 0 -j ACCEPT
sudo firewall-cmd --reload
```

### 3. 패키지 의존성 문제
```bash
# 의존성 문제 해결
sudo dnf clean all
sudo dnf makecache
```

## 추가 도구 설치

CentOS 8에서 유용한 추가 도구들:

```bash
# 개발 도구
sudo dnf groupinstall "Development Tools"

# 네트워크 도구
sudo dnf install -y net-tools bind-utils

# 시스템 모니터링
sudo dnf install -y htop iotop sysstat

# 텍스트 에디터
sudo dnf install -y vim nano
```

## 서비스 관리 명령어

```bash
# 서비스 상태 확인
sudo systemctl status docker
sudo systemctl status nginx
sudo systemctl status firewalld

# 서비스 로그 확인
sudo journalctl -u docker -f
sudo journalctl -u nginx -f

# 부팅 시 자동 시작 서비스 목록
sudo systemctl list-unit-files --state=enabled
```

---

이 가이드는 CentOS 8의 특수한 설정과 차이점을 다룹니다. 기본 설치 및 배포 과정은 [배포 가이드](./DEPLOYMENT.md)를 참조하세요.