-- 게시글 테이블 생성
CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255),
    content TEXT,
    file_name VARCHAR(255),
    file_path VARCHAR(500),
    file_size BIGINT DEFAULT 0,
    post_type VARCHAR(20) NOT NULL CHECK (post_type IN ('file', 'message')),
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ 
);

-- 업데이트 시간 자동 갱신 함수
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 트리거 생성
CREATE TRIGGER update_posts_updated_at 
    BEFORE UPDATE ON posts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_post_type ON posts(post_type);

-- 초기 데이터 (선택사항)
INSERT INTO posts (title, content, post_type) VALUES 
    ('환영합니다!', '파일 업로드 게시판에 오신 것을 환영합니다. 파일을 업로드하거나 메시지를 남겨보세요.', 'message');