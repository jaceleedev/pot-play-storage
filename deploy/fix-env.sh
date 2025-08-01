#!/bin/bash
# .env 파일 수정 스크립트

echo "🔧 환경 변수 문제를 수정합니다..."

# 서버에서 직접 실행
cd /home/pot-play-storage

# 1. 백업 생성
cp .env .env.backup.$(date +%Y%m%d_%H%M%S)

# 2. PostgreSQL 비밀번호 설정
echo "🔐 PostgreSQL 비밀번호를 설정합니다..."
sed -i 's/DB_PASSWORD=$/DB_PASSWORD=pot_storage_secure_pass_2025/' .env

# 3. Redis 비밀번호 설정
echo "🔐 Redis 비밀번호를 설정합니다..."
sed -i 's/REDIS_PASSWORD=your_redis_password_here/REDIS_PASSWORD=redis_secure_pass_2025/' .env

# 4. API 키 설정
echo "🔐 API 키를 설정합니다..."
sed -i 's/API_KEY=your-super-secret-api-key-here/API_KEY=pot_storage_api_key_2025/' .env

# 5. deploy 디렉토리에 복사
cp .env deploy/.env

# 6. migrations 디렉토리 생성
echo "📁 migrations 디렉토리를 생성합니다..."
mkdir -p deploy/migrations

echo "✅ 환경 변수 수정 완료!"
echo "🚀 이제 다시 배포를 시도해보세요."