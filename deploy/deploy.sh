#!/bin/bash
# Pot-Play-Storage 배포 스크립트
# 사용법: ./deploy.sh [command] [tag] [token]

set -e

# 색상 정의
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 변수 설정
REGISTRY="ghcr.io"
IMAGE_NAME="jaceleedev/pot-play-storage"
COMPOSE_FILE="deploy/docker-compose.prod.yml"

# 함수: 로그 출력
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

# 함수: 헬스체크
health_check() {
    local max_attempts=30
    local attempt=1
    
    log "헬스체크를 시작합니다..."
    sleep 10
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f http://localhost:8090/health > /dev/null 2>&1; then
            log "✅ 헬스체크 성공!"
            return 0
        fi
        
        warning "헬스체크 대기 중... ($attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    error "헬스체크 실패! 서비스가 정상적으로 시작되지 않았습니다."
}

# 함수: 데이터베이스 마이그레이션
run_migrations() {
    log "데이터베이스 마이그레이션을 실행합니다..."
    
    docker compose -f $COMPOSE_FILE run --rm api sh -c "
        if [ -f /app/migrations/001_initial.sql ]; then
            PGPASSWORD=\$DB_PASSWORD psql -h \$DB_HOST -U \$DB_USER -d \$DB_NAME -f /app/migrations/001_initial.sql || true
        fi
    " || warning "마이그레이션 실행 중 경고가 발생했습니다 (이미 적용된 경우 정상)"
}

# 함수: 배포 실행
deploy() {
    local tag=${1:-latest}
    local token=$2
    
    log "배포를 시작합니다. (태그: $tag)"
    
    # GitHub Container Registry 로그인
    if [ -n "$token" ]; then
        log "GitHub Container Registry에 로그인합니다..."
        echo "$token" | docker login $REGISTRY -u $USER --password-stdin
    fi
    
    # 최신 이미지 pull
    log "최신 Docker 이미지를 가져옵니다..."
    docker pull $REGISTRY/$IMAGE_NAME:$tag
    
    # docker-compose.yml 이미지 태그 업데이트
    log "docker-compose.yml을 업데이트합니다..."
    sed -i "s|image: .*pot-play-storage.*|image: $REGISTRY/$IMAGE_NAME:$tag|g" $COMPOSE_FILE
    
    # 기존 컨테이너 중지
    log "기존 서비스를 중지합니다..."
    docker compose -f $COMPOSE_FILE stop api || true
    
    # 데이터베이스 마이그레이션 실행
    run_migrations
    
    # 새 컨테이너 시작
    log "새 버전을 시작합니다..."
    docker compose -f $COMPOSE_FILE up -d
    
    # 헬스체크
    health_check
    
    # 로그 확인
    log "최근 로그를 확인합니다..."
    docker compose -f $COMPOSE_FILE logs --tail=50 api
    
    # 오래된 이미지 정리
    log "오래된 이미지를 정리합니다..."
    docker image prune -f
    
    log "✅ 배포가 성공적으로 완료되었습니다!"
}

# 함수: 롤백
rollback() {
    local previous_tag=${1:-latest}
    
    warning "이전 버전으로 롤백합니다. (태그: $previous_tag)"
    
    # 이전 이미지로 업데이트
    sed -i "s|image: .*pot-play-storage.*|image: $REGISTRY/$IMAGE_NAME:$previous_tag|g" $COMPOSE_FILE
    
    # 재시작
    docker compose -f $COMPOSE_FILE restart api
    
    # 헬스체크
    health_check
    
    log "✅ 롤백이 완료되었습니다."
}

# 함수: 상태 확인
status() {
    log "서비스 상태를 확인합니다..."
    docker compose -f $COMPOSE_FILE ps
    
    echo ""
    log "컨테이너 리소스 사용량:"
    docker stats --no-stream
}

# 메인 로직
case "$1" in
    deploy)
        deploy "$2" "$3"
        ;;
    rollback)
        rollback "$2"
        ;;
    status)
        status
        ;;
    migrate)
        run_migrations
        ;;
    *)
        echo "사용법: $0 {deploy|rollback|status|migrate} [tag] [token]"
        echo ""
        echo "Commands:"
        echo "  deploy [tag] [token]  - 새 버전 배포 (기본: latest)"
        echo "  rollback [tag]        - 이전 버전으로 롤백"
        echo "  status                - 서비스 상태 확인"
        echo "  migrate               - 데이터베이스 마이그레이션만 실행"
        exit 1
        ;;
esac