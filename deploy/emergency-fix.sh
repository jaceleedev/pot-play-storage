#!/bin/bash
# 긴급 수정 스크립트 - 서버에서 직접 실행

set -e

echo "🚨 긴급 배포 문제 해결 스크립트"

# 1. 모든 pot-storage 컨테이너 강제 중지
echo "🛑 모든 컨테이너를 강제 중지합니다..."
docker stop pot-storage-api pot-storage-postgres pot-storage-redis pot-storage-seaweedfs-master pot-storage-seaweedfs-volume pot-storage-seaweedfs-filer 2>/dev/null || true
docker rm pot-storage-api pot-storage-postgres pot-storage-redis pot-storage-seaweedfs-master pot-storage-seaweedfs-volume pot-storage-seaweedfs-filer 2>/dev/null || true

# 2. 네트워크 제거
echo "🔌 네트워크를 제거합니다..."
docker network rm deploy_pot-storage-network 2>/dev/null || true

# 3. 볼륨 제거
echo "🗑️ 볼륨을 제거합니다..."
docker volume rm deploy_postgres_data deploy_redis_data deploy_seaweedfs_master_data deploy_seaweedfs_volume_data deploy_seaweedfs_filer_data 2>/dev/null || true

# 4. 디렉토리 재생성
echo "📁 디렉토리를 재생성합니다..."
rm -rf /home/pot-play-storage/deploy/data
mkdir -p /home/pot-play-storage/deploy/data/{postgres,redis,seaweedfs/{master,volume,filer}}
chmod -R 777 /home/pot-play-storage/deploy/data/seaweedfs

# 5. .env 파일 확인
echo "🔍 환경 변수 확인..."
cd /home/pot-play-storage
if [ ! -f deploy/.env ]; then
    cp .env deploy/.env
fi

# 6. 서비스 시작
echo "🚀 서비스를 시작합니다..."
cd deploy
docker compose -f docker-compose.prod.yml up -d

# 7. 상태 확인
echo "📊 서비스 상태:"
sleep 10
docker compose -f docker-compose.prod.yml ps

echo "✅ 긴급 수정 완료!"