package models

import (
	"database/sql"
	"time"
)

// File 물리적 파일 정보 구조체
type File struct {
	ID        int       `json:"id"`
	FileHash  string    `json:"file_hash"`
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
}

// Post 게시글 구조체 (기본 + 관리자용 통합)
type Post struct {
	ID         int           `json:"id"`
	Title      string        `json:"title"`
	Content    string        `json:"content"`
	FileName   string        `json:"file_name"`  // 사용자가 업로드한 원본 파일명
	FileID     sql.NullInt32 `json:"file_id"`    // files 테이블 참조
	PostType   string        `json:"post_type"`
	IPAddress  string        `json:"ip_address"`
	CreatedAt  time.Time     `json:"created_at"`
	DeletedAt  sql.NullTime  `json:"deleted_at"`
	
	// 조인된 파일 정보 (파일 게시글인 경우)
	File *File `json:"file,omitempty"`
}

// 파일 크기를 MB로 반환
func (p *Post) GetFileSizeMB() float64 {
	if p.File != nil {
		return float64(p.File.FileSize) / (1024 * 1024)
	}
	return 0
}

// FileSizeMB 템플릿에서 사용하는 필드 (호환성)
func (p *Post) FileSizeMB() float64 {
	return p.GetFileSizeMB()
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
