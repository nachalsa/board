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
	Content string `json:"content" binding:"required"`
}
