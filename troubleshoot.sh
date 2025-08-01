#\!/bin/bash
# 1. Docker 컨테이너 상태 확인
echo '=== Docker 컨테이너 상태 ==='
docker-compose -f docker-compose.dev.yml ps

# 2. 컨테이너가 실행 중이 아니면 시작
echo -e '
=== Docker 컨테이너 시작 ==='
docker-compose -f docker-compose.dev.yml up -d

# 3. 잠시 대기 (서비스 초기화)
echo -e '
=== 서비스 초기화 대기 (10초) ==='
sleep 10

# 4. 서비스 상태 확인
echo -e '
=== 서비스 헬스 체크 ==='
echo 'API 서버:'
curl -s http://localhost:8090/health | jq '.' || echo 'API 서버 접속 실패'

echo -e '
SeaweedFS Master:'
curl -s http://localhost:9333/cluster/status | jq '.' || echo 'SeaweedFS Master 접속 실패'

echo -e '
SeaweedFS Filer:'
curl -s http://localhost:8888/ || echo 'SeaweedFS Filer 접속 실패'

# 5. 파일 업로드 테스트
echo -e '

=== 파일 업로드 테스트 ==='
if [ -f dog.jpg ]; then
    curl -X POST http://localhost:8090/api/v1/files         -H 'X-API-Key: your-api-key'         -F 'file=@dog.jpg'         -F 'collection=test'         | jq '.'
else
    echo 'dog.jpg 파일이 없습니다'
fi
