-- Pot Play Storage Database Schema
-- 전체 초기 스키마 (중복 제거 기능 포함)

-- UUID 확장 활성화
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. 스토리지 테이블 (고유 파일 저장)
CREATE TABLE storage (
    hash VARCHAR(64) PRIMARY KEY,
    size BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    storage_path VARCHAR(500) NOT NULL UNIQUE,
    reference_count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 스토리지 테이블 인덱스
CREATE INDEX idx_storage_size ON storage(size);
CREATE INDEX idx_storage_content_type ON storage(content_type);
CREATE INDEX idx_storage_reference_count ON storage(reference_count);

-- 2. 파일 테이블 (사용자 파일 참조)
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    storage_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- 레거시 컬럼 (하위 호환성, 점진적 마이그레이션용)
    size BIGINT,
    content_type VARCHAR(100),
    storage_path VARCHAR(500),
    checksum VARCHAR(64),
    -- 외래 키 제약
    CONSTRAINT fk_files_storage_hash 
        FOREIGN KEY (storage_hash) REFERENCES storage(hash) ON DELETE RESTRICT
);

-- 파일 테이블 인덱스
CREATE INDEX idx_files_storage_hash ON files(storage_hash);
CREATE INDEX idx_files_created_at ON files(created_at);
CREATE INDEX idx_files_checksum ON files(checksum);

-- 3. 자동 업데이트 트리거
CREATE OR REPLACE FUNCTION update_storage_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER storage_updated_at_trigger
    BEFORE UPDATE ON storage
    FOR EACH ROW
    EXECUTE FUNCTION update_storage_updated_at();