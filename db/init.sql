-- 파일 테이블 생성 (물리적 파일 저장 정보)
CREATE TABLE IF NOT EXISTS files (
    id SERIAL PRIMARY KEY,
    file_hash VARCHAR(16) UNIQUE NOT NULL, -- FNV 64bit 해시 (16자리 16진수)
    file_path VARCHAR(500) NOT NULL,       -- 실제 저장 경로
    file_size BIGINT NOT NULL,             -- 파일 크기
    mime_type VARCHAR(100),                -- MIME 타입
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 게시글 테이블 생성 (논리적 정보)
CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255),
    content TEXT,                          -- 메시지 타입에서만 사용
    file_name VARCHAR(255),                -- 사용자가 업로드한 원본 파일명
    file_id INTEGER REFERENCES files(id),  -- 파일 테이블 참조
    post_type VARCHAR(20) NOT NULL CHECK (post_type IN ('file', 'message')),
    ip_address VARCHAR(45),                -- 클라이언트 IP 주소 (admin에서 확인용)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ 
);


-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_posts_deleted_at ON posts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_posts_post_type ON posts(post_type);
CREATE INDEX IF NOT EXISTS idx_posts_file_id ON posts(file_id);
CREATE INDEX IF NOT EXISTS idx_files_hash ON files(file_hash);


-- 초기 데이터 (선택사항)
INSERT INTO posts (title, content, post_type, created_at) VALUES
    ('🌱 새싹 여러분 환영합니다! 오늘 하루도 행복하길 😊', '파일 업로드 게시판에 오신 것을 환영합니다. 파일을 업로드하거나 메시지를 남겨보세요.', 'message', '2025-08-11 17:30:00+09:00');