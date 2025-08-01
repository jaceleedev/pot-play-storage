# Nginx 설정 가이드

Pot Play Storage를 프로덕션 환경에서 배포할 때 사용하는 Nginx 설정 가이드입니다.

## 개요

Nginx는 리버스 프록시로 작동하여:
- SSL/TLS 종료 처리
- 보안 헤더 추가
- 파일 업로드/다운로드 최적화
- 로드 밸런싱 (필요시)

## 기본 설정

### 1. SSL 인증서 발급 (Let's Encrypt)

```bash
# Certbot 설치
sudo apt update
sudo apt install certbot python3-certbot-nginx

# SSL 인증서 발급
sudo certbot --nginx -d pot-storage.pot-play.com
```

### 2. Nginx 설정 파일 생성

`/etc/nginx/conf.d/pot-storage.pot-play.com.conf` 파일을 생성하고 아래 내용을 추가합니다:

```nginx
# pot-storage.pot-play.com 설정
upstream pot_play_storage {
    server 127.0.0.1:8090;
    keepalive 32;
    keepalive_requests 100;
    keepalive_timeout 60s;
}

server {
    listen 80;
    server_name pot-storage.pot-play.com;

    # HTTP → HTTPS 리다이렉트
    if ($host = pot-storage.pot-play.com) {
        return 301 https://$host$request_uri;
    }
    return 404;
}

server {
    listen 443 ssl http2;
    server_name pot-storage.pot-play.com;

    # SSL 설정 (Certbot이 자동 관리)
    ssl_certificate /etc/letsencrypt/live/pot-storage.pot-play.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/pot-storage.pot-play.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    # 보안 헤더
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
    
    # 파일 업로드 설정
    client_max_body_size 100M;
    client_body_timeout 60s;
    client_header_timeout 60s;
    send_timeout 60s;
    keepalive_timeout 65s;
    
    # 프록시 버퍼 설정
    proxy_buffering off;
    proxy_request_buffering off;
    proxy_connect_timeout 60s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;

    # 헬스체크 엔드포인트
    location /health {
        proxy_pass http://pot_play_storage;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        access_log off;
    }

    # API 엔드포인트
    location ~ ^/api/v1/files {
        proxy_pass http://pot_play_storage;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 파일 작업을 위한 특별 헤더
        proxy_set_header X-Original-URI $request_uri;
        proxy_set_header X-Original-Method $request_method;
        
        # 업로드를 위한 버퍼링 비활성화
        proxy_buffering off;
        proxy_request_buffering off;
        
        # 대용량 파일을 위한 확장 타임아웃
        proxy_connect_timeout 300s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # 메인 애플리케이션 프록시
    location / {
        proxy_pass http://pot_play_storage;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 표준 타임아웃
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Gzip 압축
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/json
        application/javascript
        application/xml+rss
        application/atom+xml
        image/svg+xml;

    # 로깅
    access_log /var/log/nginx/pot-storage.access.log;
    error_log /var/log/nginx/pot-storage.error.log warn;
}
```

### 3. 설정 적용

```bash
# 설정 테스트
sudo nginx -t

# Nginx 재시작
sudo nginx -s reload
```

## 주요 설정 설명

### SSL/TLS
- TLS 1.2와 1.3만 허용
- SSL 세션 캐싱으로 성능 향상
- Let's Encrypt 인증서 자동 갱신

### 보안 헤더
- **X-Frame-Options**: 클릭재킹 방지
- **X-Content-Type-Options**: MIME 타입 스니핑 방지
- **X-XSS-Protection**: XSS 공격 방지
- **Referrer-Policy**: 레퍼러 정보 제어
- **Content-Security-Policy**: 컨텐츠 소스 제한

### 파일 업로드 최적화
- 최대 업로드 크기: 100MB
- 버퍼링 비활성화로 메모리 사용 감소
- 대용량 파일을 위한 확장 타임아웃

### 성능 최적화
- HTTP/2 활성화
- Gzip 압축
- Keep-alive 연결 재사용
- 헬스체크 로깅 비활성화

## 문제 해결

### 502 Bad Gateway
- API 서버가 실행 중인지 확인: `docker ps`
- 포트 8090이 올바른지 확인
- 방화벽 설정 확인

### 413 Request Entity Too Large
- `client_max_body_size` 값 증가
- API 서버의 업로드 제한도 확인

### SSL 인증서 갱신
```bash
# 자동 갱신 테스트
sudo certbot renew --dry-run

# 수동 갱신
sudo certbot renew
```

## 모니터링

### 액세스 로그 확인
```bash
tail -f /var/log/nginx/pot-storage.access.log
```

### 에러 로그 확인
```bash
tail -f /var/log/nginx/pot-storage.error.log
```

### 연결 상태 확인
```bash
# 활성 연결 수
nginx -V 2>&1 | grep -o with-http_stub_status_module
```

## 추가 고려사항

### 로드 밸런싱
여러 API 서버를 운영할 경우:
```nginx
upstream pot_play_storage {
    server 127.0.0.1:8090 weight=1;
    server 127.0.0.1:8091 weight=1;
    keepalive 32;
}
```

### 속도 제한
DDoS 방지를 위한 요청 제한:
```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

location /api/ {
    limit_req zone=api burst=20 nodelay;
    # ... 기존 설정
}
```