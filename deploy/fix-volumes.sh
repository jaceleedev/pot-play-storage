#!/bin/bash
# Docker 볼륨 문제 해결 스크립트

set -e

echo "🔧 Docker 볼륨 문제를 해결합니다..."

# 1. 기존 컨테이너와 볼륨 정리
echo "📦 기존 컨테이너를 정리합니다..."
cd /home/pot-play-storage/deploy
docker compose -f docker-compose.prod.yml down -v || true

# 2. 기존 볼륨 완전 제거
echo "🗑️ 기존 Docker 볼륨을 제거합니다..."
docker volume rm deploy_postgres_data || true
docker volume rm deploy_redis_data || true
docker volume rm deploy_seaweedfs_master_data || true
docker volume rm deploy_seaweedfs_volume_data || true
docker volume rm deploy_seaweedfs_filer_data || true

# 3. 필요한 디렉토리 생성 (절대 경로로)
echo "📁 필요한 디렉토리를 생성합니다..."
mkdir -p /home/pot-play-storage/deploy/data/postgres
mkdir -p /home/pot-play-storage/deploy/data/redis
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/master
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/volume
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/filer
mkdir -p /home/pot-play-storage/deploy/uploads
mkdir -p /home/pot-play-storage/deploy/backups
mkdir -p /home/pot-play-storage/deploy/configs

# 4. 권한 설정
echo "🔐 디렉토리 권한을 설정합니다..."
chmod -R 755 /home/pot-play-storage/deploy/data
chmod -R 777 /home/pot-play-storage/deploy/data/seaweedfs
chmod -R 755 /home/pot-play-storage/deploy/uploads
chmod -R 755 /home/pot-play-storage/deploy/backups

# 5. 디렉토리 확인
echo "✅ 생성된 디렉토리 확인:"
ls -la /home/pot-play-storage/deploy/data/
ls -la /home/pot-play-storage/deploy/data/seaweedfs/

echo "🎯 볼륨 준비가 완료되었습니다!"