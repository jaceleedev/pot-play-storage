#!/bin/bash
# Pot Play Storage - 통합 배포 스크립트
set -e

# =============================================================================
# 환경 설정
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_NAME="pot-play-storage"
COMPOSE_FILE="docker-compose.prod.yml"
ENV_FILE=".env"
GHCR_REGISTRY="ghcr.io"

# 색상 코드
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# =============================================================================
# 도움말
# =============================================================================
show_help() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  deploy       프로덕션 배포 (기본값)"
    echo "  rollback     이전 버전으로 롤백"
    echo "  status       서비스 상태 확인"
    echo "  fix          문제 해결 (재시작)"
    echo "  clean        전체 초기화 및 재배포"
    echo "  migrate      데이터베이스 마이그레이션만 실행"
    echo ""
    echo "Options:"
    echo "  --skip-health   헬스체크 건너뛰기"
    echo "  --no-pull       이미지 pull 건너뛰기"
    echo "  -h, --help      이 도움말 표시"
}

# =============================================================================
# 유틸리티 함수
# =============================================================================
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

check_prerequisites() {
    log_info "필수 구성 요소 확인 중..."
    
    # Docker 확인
    if ! command -v docker &> /dev/null; then
        log_error "Docker가 설치되어 있지 않습니다."
        exit 1
    fi
    
    # Docker Compose 확인
    if ! docker compose version &> /dev/null; then
        log_error "Docker Compose가 설치되어 있지 않습니다."
        exit 1
    fi
    
    # 환경 파일 확인
    if [ ! -f "$ENV_FILE" ]; then
        log_error ".env 파일이 없습니다. .env.example을 복사하여 설정하세요."
        exit 1
    fi
    
    # docker-compose.prod.yml 확인
    if [ ! -f "$COMPOSE_FILE" ]; then
        log_error "$COMPOSE_FILE 파일이 없습니다."
        exit 1
    fi
}

create_directories() {
    log_info "필요한 디렉토리 생성 중..."
    
    # 데이터 디렉토리 생성
    mkdir -p data/{postgres,redis,seaweedfs/{master,volume,filer}}
    mkdir -p uploads backups
    
    # SeaweedFS 권한 설정
    chmod -R 777 data/seaweedfs
    
    log_info "디렉토리 생성 완료"
}

health_check() {
    if [ "$SKIP_HEALTH" = true ]; then
        log_warning "헬스체크를 건너뜁니다."
        return 0
    fi
    
    log_info "서비스 헬스체크 시작..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if docker compose -f "$COMPOSE_FILE" ps | grep -q "healthy"; then
            log_info "헬스체크 성공! (시도: $attempt/$max_attempts)"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    log_error "헬스체크 실패. 로그를 확인하세요."
    docker compose -f "$COMPOSE_FILE" logs --tail=50
    return 1
}

# =============================================================================
# 배포 명령
# =============================================================================
deploy() {
    log_info "Pot Play Storage 배포 시작..."
    
    # GitHub Container Registry 로그인
    if [ -n "$GITHUB_TOKEN" ]; then
        log_info "GitHub Container Registry 로그인 중..."
        echo "$GITHUB_TOKEN" | docker login $GHCR_REGISTRY -u "$GITHUB_ACTOR" --password-stdin
    fi
    
    # 이미지 Pull
    if [ "$NO_PULL" != true ]; then
        log_info "최신 이미지 다운로드 중..."
        docker compose -f "$COMPOSE_FILE" pull
    fi
    
    # 기존 서비스 중지
    log_info "기존 서비스 중지 중..."
    docker compose -f "$COMPOSE_FILE" down
    
    # 디렉토리 생성
    create_directories
    
    # 서비스 시작
    log_info "서비스 시작 중..."
    docker compose -f "$COMPOSE_FILE" up -d
    
    # 헬스체크
    health_check
    
    log_info "배포 완료!"
    docker compose -f "$COMPOSE_FILE" ps
}

rollback() {
    log_info "이전 버전으로 롤백 중..."
    
    # 현재 버전 백업
    docker compose -f "$COMPOSE_FILE" down
    
    # 이전 이미지로 재시작 (태그 관리 필요)
    log_warning "롤백 기능은 이미지 태그 관리가 필요합니다."
    log_info "수동으로 이전 버전의 이미지 태그를 지정하여 재배포하세요."
}

status() {
    log_info "서비스 상태 확인..."
    docker compose -f "$COMPOSE_FILE" ps
    echo ""
    log_info "컨테이너 상태:"
    docker ps -a | grep "$PROJECT_NAME" || true
}

fix() {
    log_info "서비스 재시작 중..."
    docker compose -f "$COMPOSE_FILE" restart
    sleep 5
    health_check
}

clean() {
    log_warning "전체 시스템을 초기화합니다. 데이터가 삭제될 수 있습니다!"
    read -p "정말 계속하시겠습니까? (yes/no): " -n 3 -r
    echo
    if [[ ! $REPLY =~ ^yes$ ]]; then
        log_info "취소되었습니다."
        exit 0
    fi
    
    log_info "전체 초기화 시작..."
    
    # 모든 서비스 중지
    docker compose -f "$COMPOSE_FILE" down -v
    
    # 볼륨 제거
    log_info "Docker 볼륨 제거 중..."
    docker volume rm $(docker volume ls -q | grep "$PROJECT_NAME") 2>/dev/null || true
    
    # 데이터 디렉토리 초기화
    log_info "데이터 디렉토리 초기화 중..."
    rm -rf data/*
    
    # 재배포
    create_directories
    deploy
}

migrate() {
    log_info "데이터베이스 마이그레이션 실행 중..."
    
    if [ ! -d "migrations" ]; then
        log_error "migrations 디렉토리가 없습니다."
        exit 1
    fi
    
    # PostgreSQL 컨테이너가 실행 중인지 확인
    if ! docker ps | grep -q "pot-storage-postgres"; then
        log_error "PostgreSQL 컨테이너가 실행 중이지 않습니다."
        exit 1
    fi
    
    # 마이그레이션 실행
    for migration in migrations/*.sql; do
        if [ -f "$migration" ]; then
            log_info "마이그레이션 실행: $(basename "$migration")"
            docker exec -i pot-storage-postgres psql -U pot_storage_user -d pot_storage_db < "$migration"
        fi
    done
    
    log_info "마이그레이션 완료!"
}

# =============================================================================
# 메인 실행 부분
# =============================================================================
# 옵션 파싱
SKIP_HEALTH=false
NO_PULL=false
COMMAND="deploy"

while [[ $# -gt 0 ]]; do
    case $1 in
        deploy|rollback|status|fix|clean|migrate)
            COMMAND=$1
            shift
            ;;
        --skip-health)
            SKIP_HEALTH=true
            shift
            ;;
        --no-pull)
            NO_PULL=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "알 수 없는 옵션: $1"
            show_help
            exit 1
            ;;
    esac
done

# 필수 구성 요소 확인
check_prerequisites

# 명령 실행
case $COMMAND in
    deploy)
        deploy
        ;;
    rollback)
        rollback
        ;;
    status)
        status
        ;;
    fix)
        fix
        ;;
    clean)
        clean
        ;;
    migrate)
        migrate
        ;;
    *)
        log_error "알 수 없는 명령: $COMMAND"
        show_help
        exit 1
        ;;
esac