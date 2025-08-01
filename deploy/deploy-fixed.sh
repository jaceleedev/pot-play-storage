#!/bin/bash
# 수정된 배포 스크립트 - 볼륨 문제 해결 포함

set -e

# 색상 정의
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 0. 작업 디렉토리로 이동
cd /home/pot-play-storage

# 1. 최신 코드 pull
log "🔄 최신 코드를 가져옵니다..."
git pull origin main

# 2. 환경 변수 확인 및 복사
if [ ! -f .env ]; then
    error "❌ .env 파일이 없습니다!"
fi
cp .env deploy/.env

# 3. 기존 서비스 정리
log "🛑 기존 서비스를 정리합니다..."
cd deploy
docker compose -f docker-compose.prod.yml down || true

# 4. Docker 볼륨 정리
log "🗑️ 기존 Docker 볼륨을 제거합니다..."
docker volume rm deploy_postgres_data deploy_redis_data deploy_seaweedfs_master_data deploy_seaweedfs_volume_data deploy_seaweedfs_filer_data 2>/dev/null || true

# 5. 디렉토리 구조 생성 (절대 경로)
log "📁 필요한 디렉토리를 생성합니다..."
mkdir -p /home/pot-play-storage/deploy/data/postgres
mkdir -p /home/pot-play-storage/deploy/data/redis
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/master
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/volume
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/filer
mkdir -p /home/pot-play-storage/deploy/uploads
mkdir -p /home/pot-play-storage/deploy/backups
mkdir -p /home/pot-play-storage/deploy/configs

# 6. 권한 설정
log "🔐 디렉토리 권한을 설정합니다..."
chmod -R 755 /home/pot-play-storage/deploy/data
chmod -R 777 /home/pot-play-storage/deploy/data/seaweedfs
chmod -R 755 /home/pot-play-storage/deploy/uploads
chmod -R 755 /home/pot-play-storage/deploy/backups

# 7. GHCR 로그인
log "🔐 GitHub Container Registry 로그인..."
echo "${{ github.token }}" | docker login ghcr.io -u jaceleedev --password-stdin

# 8. 최신 이미지 pull
log "📦 최신 Docker 이미지를 가져옵니다..."
docker pull ghcr.io/jaceleedev/pot-play-storage:latest

# 9. docker-compose.yml 이미지 태그 업데이트
log "📝 docker-compose.yml 업데이트..."
sed -i "s|image: .*pot-play-storage.*|image: ghcr.io/jaceleedev/pot-play-storage:latest|g" docker-compose.prod.yml

# 10. 서비스 시작
log "🚀 새 버전을 시작합니다..."
docker compose -f docker-compose.prod.yml up -d

# 11. 헬스 체크
log "❤️ 헬스 체크 중..."
sleep 15
for i in {1..30}; do
    if curl -f http://localhost:8090/health 2>/dev/null; then
        log "✅ 배포가 성공적으로 완료되었습니다!"
        break
    fi
    warning "⏳ 서비스가 시작되기를 기다리는 중... ($i/30)"
    sleep 3
done

# 12. 컨테이너 상태 확인
log "📊 컨테이너 상태:"
docker compose -f docker-compose.prod.yml ps

# 13. 로그 확인
log "📜 최근 로그:"
docker compose -f docker-compose.prod.yml logs --tail=20

# 14. 오래된 이미지 정리
log "🧹 오래된 이미지를 정리합니다..."
docker image prune -f

log "🎉 배포 프로세스가 완료되었습니다!"