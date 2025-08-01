# 🚀 Pot Storage 배포 가이드

**이전**: [기술 상세](./TECHNICAL.md) | **다음**: [README](./README.md)

Pot Storage 서비스를 프로덕션 환경에 배포하기 위한 단계별 가이드입니다.

## 📋 사전 요구사항

- Ubuntu 20.04+ 또는 유사한 Linux 배포판
- Docker 및 Docker Compose 설치
- Nginx 설치  
- 도메인 이름 설정 (pot-storage.pot-play.com)
- SSL 인증서 (Let's Encrypt 권장)
- 최소 4GB RAM, 2 CPU 코어, 50GB 스토리지

## ⚡ 빠른 시작

```bash
# 1. 저장소 클론
git clone https://github.com/your-org/pot-play-storage.git
cd pot-play-storage

# 2. 환경 설정
cp deploy/.env.example .env
# .env 파일을 본인의 설정에 맞게 편집

# 3. 데이터 디렉토리 생성
sudo mkdir -p /opt/pot-storage/{data,uploads,backups}
sudo chown -R $USER:$USER /opt/pot-storage

# 4. 배포 실행
docker compose -f deploy/docker-compose.prod.yml up -d
```

## 🔧 일회성 서버 초기 설정

새 서버에 처음 배포할 때 수행해야 하는 일회성 설정 단계입니다. 이 작업들은 GitHub Actions 외부에서 수동으로 한 번만 실행하면 됩니다.

### 1. 기본 시스템 설정

```bash
# 루트 사용자로 로그인하여 시스템 업데이트
apt update && apt upgrade -y

# 필수 패키지 설치
apt install -y curl wget git unzip

# 시간대 설정 (한국 시간)
timedatectl set-timezone Asia/Seoul
```

### 2. Docker 설치 (CentOS 8)

```bash
# CentOS 8용 Docker 설치
# 기존 Docker 관련 패키지 제거
sudo dnf remove docker \
                docker-client \
                docker-client-latest \
                docker-common \
                docker-latest \
                docker-latest-logrotate \
                docker-logrotate \
                docker-engine

# 필요한 패키지 설치
sudo dnf install -y dnf-plugins-core

# Docker 저장소 추가
sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

# Docker Engine 설치
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Docker 서비스 시작 및 자동 시작 설정
sudo systemctl start docker
sudo systemctl enable docker

# Docker 설치 확인
docker --version
docker compose version  # v2는 'docker compose' 명령 사용

# root 사용자로 Docker 실행 권한 확인
docker run hello-world
```

### 3. 프로젝트 디렉토리 초기 설정

```bash
# 프로젝트 루트 디렉토리 생성
mkdir -p /opt/pot-storage
cd /opt/pot-storage

# 저장소 최초 클론 (GitHub Actions가 이후 업데이트)
git clone https://github.com/your-org/pot-play-storage.git .

# 데이터 디렉토리 구조 생성
mkdir -p data/{postgres,redis,seaweedfs/{master,volume,filer}}
mkdir -p uploads backups logs

# 적절한 권한 설정
chmod -R 755 data uploads backups logs
chown -R 999:999 data/postgres
chown -R 999:999 data/redis
```

### 4. 환경 설정 파일 초기 설정

```bash
# 환경 설정 파일 생성
cp deploy/.env.example .env

# .env 파일을 프로덕션 값으로 편집
nano .env
```

**.env 파일 필수 설정 예시**:
```bash
# 데이터베이스 설정
DB_USER=pot_storage_user
DB_PASSWORD=your_secure_db_password_here
DB_NAME=pot_storage_prod

# Redis 설정
REDIS_PASSWORD=your_secure_redis_password_here

# 보안 키
API_KEY=your-super-secret-api-key-here
JWT_SECRET=your-jwt-secret-key-here

# 스토리지 경로 (절대 경로)
POSTGRES_DATA_PATH=/opt/pot-storage/data/postgres
REDIS_DATA_PATH=/opt/pot-storage/data/redis
UPLOAD_VOLUME_PATH=/opt/pot-storage/uploads

# 성능 및 로깅
LOG_LEVEL=warn
REDIS_MAX_MEMORY=512mb
```

### 5. 네트워크 및 보안 설정 (CentOS 8)

```bash
# CentOS 8은 firewalld 사용
# firewalld 상태 확인 및 시작
sudo systemctl status firewalld
sudo systemctl start firewalld
sudo systemctl enable firewalld

# 기본 존 확인
sudo firewall-cmd --get-default-zone

# SSH 서비스 허용 (이미 기본적으로 허용됨)
sudo firewall-cmd --permanent --add-service=ssh

# HTTP/HTTPS 서비스 허용
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https

# 방화벽 규칙 다시 로드
sudo firewall-cmd --reload

# 현재 활성화된 서비스 확인
sudo firewall-cmd --list-all

# SELinux 상태 확인 (CentOS 기본 보안)
sudo getenforce

# SELinux가 Docker와 충돌하는 경우 Permissive 모드로 설정
# sudo setenforce 0
# 영구 설정: /etc/selinux/config 파일에서 SELINUX=permissive
```

### 6. Nginx 설치 및 기본 설정 (CentOS 8)

```bash
# EPEL 저장소 설치 (이미 설치되어 있을 수 있음)
sudo dnf install -y epel-release

# Nginx 설치
sudo dnf install -y nginx

# Nginx 서비스 활성화
systemctl enable nginx
systemctl start nginx

# 기본 웹페이지 확인
curl -I http://localhost
```

### 7. SSL 인증서 준비 (Let's Encrypt)

```bash
# Certbot 설치
apt install -y certbot python3-certbot-nginx

# DNS가 올바르게 설정되었는지 확인 후 인증서 발급
# (도메인이 현재 서버 IP를 가리키고 있어야 함)
certbot --nginx -d pot-storage.pot-play.com

# 자동 갱신 설정
echo "0 12 * * * /usr/bin/certbot renew --quiet" | crontab -
```

### 8. GitHub Actions를 위한 사전 테스트

```bash
# Docker 이미지 로그인 테스트 (GitHub Actions에서 사용할 토큰으로)
echo "YOUR_GITHUB_TOKEN" | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# 기본 서비스 시작 테스트
docker-compose -f deploy/docker-compose.prod.yml up -d

# 서비스 상태 확인
docker-compose -f deploy/docker-compose.prod.yml ps

# 테스트 후 정리
docker-compose -f deploy/docker-compose.prod.yml down
```

### 9. 로그 로테이션 설정

```bash
# 로그 로테이션 설정 파일 생성
cat > /etc/logrotate.d/pot-storage << 'EOF'
/opt/pot-storage/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 root root
    postrotate
        systemctl reload nginx
    endscript
}
EOF
```

### 10. 시스템 모니터링 기본 설정

```bash
# 시스템 리소스 모니터링을 위한 htop 설치
apt install -y htop iotop

# 디스크 사용량 모니터링 명령어 확인
df -h
free -h
```

---

**⚠️ 중요 사항**:
- 이 초기 설정은 **서버당 한 번만** 실행하면 됩니다
- `.env` 파일의 비밀번호와 키는 반드시 안전한 값으로 변경하세요
- GitHub Actions secrets에 `SSH_HOST`와 `SSH_ROOT_PASSWORD`를 설정해야 합니다
- 도메인 DNS 설정이 완료된 후 SSL 인증서를 발급하세요

초기 설정 완료 후에는 GitHub Actions가 자동으로 배포를 처리합니다.

## 📝 상세 배포 단계

### 1. 서버 준비 (CentOS 8)

#### 시스템 패키지 업데이트
```bash
sudo dnf update -y
```

#### Docker 설치 (CentOS 8)
```bash
# 기존 Docker 패키지 제거
sudo dnf remove docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine

# 필요한 패키지 설치
sudo dnf install -y dnf-plugins-core

# Docker 저장소 추가
sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

# Docker Engine 설치
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Docker 서비스 시작
sudo systemctl start docker
sudo systemctl enable docker
```

#### Nginx 설치 (CentOS 8)
```bash
# EPEL 저장소 설치
sudo dnf install -y epel-release

# Nginx 설치
sudo dnf install -y nginx
sudo systemctl enable nginx
sudo systemctl start nginx
```

### 2. 애플리케이션 설정

#### 애플리케이션 디렉토리 생성
```bash
sudo mkdir -p /opt/pot-storage
sudo chown -R $USER:$USER /opt/pot-storage
cd /opt/pot-storage
```

#### 저장소 클론
```bash
git clone https://github.com/your-org/pot-play-storage.git .
```

#### 환경 설정 파일 구성
```bash
cp deploy/.env.example .env
```

`.env` 파일을 프로덕션 값으로 편집:
```bash
nano .env
```

### 3. SSL 인증서 설정

#### Certbot 설치 (CentOS 8)
```bash
sudo dnf install -y certbot python3-certbot-nginx
```

#### SSL 인증서 발급
```bash
sudo certbot --nginx -d pot-storage.pot-play.com
```

#### 자동 갱신 확인 (CentOS 8)
```bash
# CentOS 8은 systemd timer를 사용하여 자동 갱신
sudo systemctl status certbot-renew.timer
sudo systemctl enable certbot-renew.timer
```

### 4. Nginx 설정

#### nginx 설정 복사 (CentOS 8)
```bash
# CentOS 8은 sites-available/enabled 대신 conf.d 사용
sudo cp deploy/nginx/pot-storage-complete.conf /etc/nginx/conf.d/pot-storage.pot-play.com.conf
```

#### nginx 테스트 및 재로드
```bash
sudo nginx -t
sudo systemctl reload nginx
```

### 5. 데이터베이스 및 스토리지 설정

#### 데이터 디렉토리 생성
```bash
mkdir -p data/{postgres,redis,seaweedfs/{master,volume,filer}}
mkdir -p uploads backups
chmod 755 data uploads backups
```

#### 적절한 권한 설정
```bash
# PostgreSQL 데이터 디렉토리
sudo chown -R 999:999 data/postgres

# Redis 데이터 디렉토리  
sudo chown -R 999:999 data/redis

# 업로드 디렉토리
sudo chown -R 1000:1000 uploads
```

### 6. 애플리케이션 배포

#### 서비스 시작
```bash
# 환경 변수 로드
export $(cat .env | grep -v '^#' | xargs)

# 서비스 시작
docker compose -f deploy/docker-compose.prod.yml up -d
```

#### 배포 검증
```bash
# 서비스 상태 확인
docker compose -f deploy/docker-compose.prod.yml ps

# 로그 확인
docker compose -f deploy/docker-compose.prod.yml logs -f api

# 헬스 엔드포인트 테스트
curl http://localhost:8090/health
```

### 7. 데이터베이스 마이그레이션

```bash
# 마이그레이션 실행 (해당하는 경우)
docker compose -f deploy/docker-compose.prod.yml exec api sh -c '
  # 여기에 마이그레이션 명령 추가
  echo "마이그레이션 완료"
'
```

## ⚙️ 환경 변수 설정

### 필수 변수

```bash
# 데이터베이스 설정
DB_USER=your_db_user
DB_PASSWORD=your_secure_password
DB_NAME=pot_storage_prod

# Redis 설정  
REDIS_PASSWORD=your_redis_password

# 보안
API_KEY=your-super-secret-api-key
JWT_SECRET=your-jwt-secret-key

# 스토리지 경로
POSTGRES_DATA_PATH=/opt/pot-storage/data/postgres
REDIS_DATA_PATH=/opt/pot-storage/data/redis
UPLOAD_VOLUME_PATH=/opt/pot-storage/uploads
```

### 선택적 변수

```bash
# SeaweedFS 설정
SEAWEEDFS_VOLUME_SIZE_LIMIT=2000
SEAWEEDFS_REPLICATION=001
SEAWEEDFS_MAX_VOLUMES=200
SEAWEEDFS_COLLECTION=pot-storage-prod

# 성능 튜닝
REDIS_MAX_MEMORY=512mb
SERVER_READ_TIMEOUT=60s
SERVER_WRITE_TIMEOUT=300s

# 로깅
LOG_LEVEL=warn
LOG_FORMAT=json
```

## 🔄 GitHub Actions 설정

### 필수 시크릿

GitHub 저장소에서 다음 시크릿을 설정하세요:

```bash
# 서버 접근 (루트 계정 사용)
SSH_HOST=your.server.ip.address
SSH_ROOT_PASSWORD=your_root_password

# 컨테이너 레지스트리
GITHUB_TOKEN=automatically_provided

# 선택사항: 알림
SLACK_WEBHOOK_URL=your_slack_webhook_url
```

**시크릿 설정 방법**:
1. GitHub 저장소 → Settings → Secrets and variables → Actions
2. "New repository secret" 클릭하여 각 시크릿 추가
3. `SSH_HOST`: 서버의 공인 IP 주소 (예: 192.168.1.100)
4. `SSH_ROOT_PASSWORD`: 서버 루트 계정 비밀번호

### 배포 프로세스

1. main 브랜치에 푸시하면 워크플로우 트리거
2. PostgreSQL 및 Redis 서비스로 테스트 실행
3. Docker 이미지 빌드 후 GitHub Container Registry에 푸시
4. 프로덕션 서버에 애플리케이션 배포
5. 헬스 체크로 배포 성공 검증
6. 설정된 채널로 알림 전송

## 📊 모니터링 및 유지보수

### 헬스 체크

```bash
# 애플리케이션 헬스
curl https://pot-storage.pot-play.com/health

# 데이터베이스 연결
docker compose -f deploy/docker-compose.prod.yml exec postgres pg_isready -U $DB_USER

# Redis 연결
docker compose -f deploy/docker-compose.prod.yml exec redis redis-cli ping

# SeaweedFS 마스터
curl http://localhost:9333/cluster/status
```

### 로그 관리

```bash
# 애플리케이션 로그 보기
docker compose -f deploy/docker-compose.prod.yml logs -f api

# nginx 로그 보기
sudo tail -f /var/log/nginx/pot-storage.access.log
sudo tail -f /var/log/nginx/pot-storage.error.log

# 시스템 로그 보기
journalctl -u docker -f
```

### 백업 절차

#### 데이터베이스 백업
```bash
# 백업 생성
docker compose -f deploy/docker-compose.prod.yml exec postgres pg_dump -U $DB_USER $DB_NAME > backups/backup_$(date +%Y%m%d_%H%M%S).sql

# 자동 백업 스크립트
cat << 'EOF' > /opt/pot-storage/backup.sh
#!/bin/bash
BACKUP_DIR="/opt/pot-storage/backups"
DATE=$(date +%Y%m%d_%H%M%S)
docker compose -f /opt/pot-storage/deploy/docker-compose.prod.yml exec -T postgres pg_dump -U $DB_USER $DB_NAME > $BACKUP_DIR/backup_$DATE.sql
gzip $BACKUP_DIR/backup_$DATE.sql
find $BACKUP_DIR -name "*.gz" -mtime +7 -delete
EOF

chmod +x /opt/pot-storage/backup.sh

# crontab에 추가
echo "0 2 * * * /opt/pot-storage/backup.sh" | crontab -
```

#### 파일 스토리지 백업
```bash
# 업로드 디렉토리 백업
rsync -av --delete /opt/pot-storage/uploads/ /opt/pot-storage/backups/uploads/

# SeaweedFS 데이터 백업
rsync -av --delete /opt/pot-storage/data/seaweedfs/ /opt/pot-storage/backups/seaweedfs/
```

### 업데이트 및 롤백

#### 애플리케이션 업데이트
```bash
cd /opt/pot-storage
git pull origin main
docker compose -f deploy/docker-compose.prod.yml pull
docker compose -f deploy/docker-compose.prod.yml up -d
```

#### 배포 롤백
```bash
# 현재 서비스 중지
docker compose -f deploy/docker-compose.prod.yml down

# 이전 버전으로 전환
git checkout previous-commit-hash

# 이전 버전으로 시작
docker compose -f deploy/docker-compose.prod.yml up -d
```

## 🔒 보안 고려사항

### 방화벽 설정
```bash
# SSH, HTTP, HTTPS 허용
sudo ufw allow ssh
sudo ufw allow 80
sudo ufw allow 443

# 서비스 직접 접근 차단
sudo ufw deny 5432  # PostgreSQL
sudo ufw deny 6379  # Redis
sudo ufw deny 8090  # Application
sudo ufw deny 9333  # SeaweedFS

sudo ufw enable
```

### 파일 권한
```bash
# 보안 권한 설정
chmod 600 .env
chmod 700 data/
chmod 755 uploads/
```

### 정기 보안 업데이트
```bash
# 시스템 패키지 업데이트
sudo apt update && sudo apt upgrade -y

# Docker 이미지 업데이트
docker compose -f deploy/docker-compose.prod.yml pull
docker compose -f deploy/docker-compose.prod.yml up -d
```

## ⚡ 성능 최적화

### Nginx 최적화
```bash
# nginx 설정 편집
sudo nano /etc/nginx/nginx.conf

# 다음 최적화 추가:
worker_processes auto;
worker_connections 2048;
sendfile on;
tcp_nopush on;
tcp_nodelay on;
```

### 데이터베이스 최적화
```bash
# PostgreSQL 튜닝 (사용 가능한 RAM에 따라 조정)
# postgresql.conf에 추가:
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
```

### Redis 최적화
```bash
# docker-compose에서 Redis 튜닝
REDIS_MAX_MEMORY=512mb
```

## 🔧 문제 해결

### 일반적인 문제

#### 서비스가 시작되지 않음
```bash
# 로그 확인
docker compose -f deploy/docker-compose.prod.yml logs api

# 포트 충돌 확인
sudo netstat -tulpn | grep :8090

# 디스크 공간 확인
df -h
```

#### 데이터베이스 연결 문제
```bash
# PostgreSQL 상태 확인
docker compose -f deploy/docker-compose.prod.yml exec postgres pg_isready

# 환경 변수 확인
docker compose -f deploy/docker-compose.prod.yml exec api env | grep DB_
```

#### 파일 업로드 문제
```bash
# 업로드 디렉토리 권한 확인
ls -la uploads/

# nginx client_max_body_size 확인
sudo nginx -T | grep client_max_body_size

# 디스크 공간 확인
df -h /opt/pot-storage/
```

#### SSL 인증서 문제
```bash
# 인증서 상태 확인
sudo certbot certificates

# SSL 설정 테스트
openssl s_client -connect pot-storage.pot-play.com:443
```

### 성능 문제
```bash
# 리소스 사용량 모니터링
docker stats

# 애플리케이션 메트릭 확인
curl https://pot-storage.pot-play.com/health

# 에러 로그 모니터링
docker compose -f deploy/docker-compose.prod.yml logs -f api | grep ERROR
```

### 복구 절차

#### 전체 시스템 복구
```bash
# 1. 모든 서비스 중지
docker compose -f deploy/docker-compose.prod.yml down

# 2. 데이터베이스 복원
docker compose -f deploy/docker-compose.prod.yml up -d postgres
gunzip < backups/backup_latest.sql.gz | docker compose -f deploy/docker-compose.prod.yml exec -T postgres psql -U $DB_USER $DB_NAME

# 3. 파일 업로드 복원
rsync -av backups/uploads/ uploads/

# 4. 모든 서비스 시작
docker compose -f deploy/docker-compose.prod.yml up -d
```

## 🛠️ 지원 및 유지보수

### 정기 유지보수 작업

1. **주간**: 로그 확인, 디스크 사용량 모니터링, 백업 검증
2. **월간**: 시스템 패키지 업데이트, 보안 로그 검토
3. **분기별**: Docker 이미지 업데이트, 성능 메트릭 검토

### 연락처 정보

- **문서**: [프로젝트 저장소](https://github.com/your-org/pot-play-storage)
- **이슈**: [GitHub Issues](https://github.com/your-org/pot-play-storage/issues)
- **지원**: support@pot-play.com

---

**마지막 업데이트**: 2025-08-01  
**버전**: 1.0.0

**이전**: [기술 상세](./TECHNICAL.md) | **다음**: [README](./README.md)