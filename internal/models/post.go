package models

import (
	"database/sql"
	"time"
)

// Post 게시글 구조체 (기본 + 관리자용 통합)
type Post struct {
	ID         int           `json:"id"`
	Title      string        `json:"title"`
	Content    string        `json:"content"`
	FileName   string        `json:"file_name"`
	FilePath   string        `json:"file_path"`
	FileSize   int64         `json:"file_size"`
	FileSizeMB float64       `json:"file_size_mb"`
	PostType   string        `json:"post_type"`
	IPAddress  string        `json:"ip_address"`
	CreatedAt  time.Time     `json:"created_at"`
	DeletedAt  sql.NullTime  `json:"deleted_at"`
}

// 한국 시간대로 변환된 시간 반환
func (p *Post) GetKSTTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Seoul")
	return p.CreatedAt.In(loc)
}

// 삭제 시간을 한국 시간대로 변환
func (p *Post) GetDeletedKSTTime() *time.Time {
	if !p.DeletedAt.Valid {
		return nil
	}
	loc, _ := time.LoadLocation("Asia/Seoul")
	t := p.DeletedAt.Time.In(loc)
	return &t
}

// PostStats 게시글 통계
type PostStats struct {
	TotalPosts   int `json:"total_posts"`
	FilePosts    int `json:"file_posts"`
	MessagePosts int `json:"message_posts"`
}

// FileUploadRequest 파일 업로드 요청
type FileUploadRequest struct {
	Title string
	File  interface{}
}

// MessageUploadRequest 메시지 업로드 요청
type MessageUploadRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
}
