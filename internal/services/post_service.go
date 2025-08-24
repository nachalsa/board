package services

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"file-board/internal/config"
	"file-board/internal/models"
)

type PostService struct {
	db  *sql.DB
	cfg *config.Config
}

func NewPostService(db *sql.DB, cfg *config.Config) *PostService {
	return &PostService{db: db, cfg: cfg}
}

// GetPosts 게시글 목록 조회 (일반 사용자용)
func (s *PostService) GetPosts() ([]models.Post, error) {
	rows, err := s.db.Query(`
		SELECT id, title, content, file_name, file_path, file_size, post_type, created_at 
		FROM posts 
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		post, err := s.scanPost(rows, false)
		if err != nil {
			continue
		}
		posts = append(posts, *post)
	}

	return posts, nil
}

// GetAllPosts 모든 게시글 조회 (관리자용)
func (s *PostService) GetAllPosts() ([]models.Post, error) {
	rows, err := s.db.Query(`
		SELECT id, title, content, file_name, file_path, file_size, post_type, ip_address, created_at, deleted_at 
		FROM posts 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		post, err := s.scanPost(rows, true)
		if err != nil {
			continue
		}
		posts = append(posts, *post)
	}

	return posts, nil
}

// GetPostFile 파일 정보 조회
func (s *PostService) GetPostFile(id int) (string, string, error) {
	var fileName, filePath string
	err := s.db.QueryRow(`
		SELECT file_name, file_path 
		FROM posts 
		WHERE id = $1 AND post_type = 'file'
	`, id).Scan(&fileName, &filePath)
	
	return fileName, filePath, err
}

// CreateFilePost 파일 게시글 생성
func (s *PostService) CreateFilePost(title string, file *multipart.FileHeader, ipAddress string) error {
	// 파일 크기 확인
	if file.Size > s.cfg.File.MaxFileSize {
		return fmt.Errorf("파일 크기는 %s를 초과할 수 없습니다", s.cfg.GetMaxFileSizeText())
	}

	// 업로드 디렉토리 생성
	if err := os.MkdirAll(s.cfg.File.UploadsDir, 0755); err != nil {
		return fmt.Errorf("업로드 디렉토리 생성 실패: %v", err)
	}

	// 중복 파일명 처리
	fileName := s.getUniqueFileName(s.cfg.File.UploadsDir, file.Filename)
	filePath := filepath.Join(s.cfg.File.UploadsDir, fileName)

	// 파일 저장
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("파일 열기 실패: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("파일 생성 실패: %v", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return fmt.Errorf("파일 복사 실패: %v", err)
	}

	// 제목이 없으면 파일명 사용
	if title == "" {
		title = file.Filename
	}

	// 데이터베이스에 저장
	_, err = s.db.Exec(`
		INSERT INTO posts (title, file_name, file_path, file_size, post_type, ip_address) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`, title, fileName, filePath, file.Size, "file", ipAddress)

	return err
}

// CreateMessagePost 메시지 게시글 생성
func (s *PostService) CreateMessagePost(title, content, ipAddress string) error {
	_, err := s.db.Exec(`
		INSERT INTO posts (title, content, post_type, ip_address) 
		VALUES ($1, $2, $3, $4)
	`, title, content, "message", ipAddress)
	
	return err
}

// DeletePost 게시글 삭제 (소프트 삭제)
func (s *PostService) DeletePost(id int) error {
	// 트랜잭션 시작
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %v", err)
	}
	defer tx.Rollback()

	// 게시글 정보 조회
	var fileName, filePath, postType string
	err = tx.QueryRow(`
		SELECT COALESCE(file_name, ''), COALESCE(file_path, ''), post_type
		FROM posts 
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&fileName, &filePath, &postType)
	
	if err != nil {
		return fmt.Errorf("게시글을 찾을 수 없습니다: %v", err)
	}

	// 게시글을 삭제됨으로 표시
	_, err = tx.Exec(`
		UPDATE posts 
		SET deleted_at = NOW() 
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("게시글 삭제 표시 실패: %v", err)
	}

	// 파일 게시글인 경우 파일 이동
	if postType == "file" && filePath != "" {
		if err := s.moveFileToDeleted(filePath, fileName, tx, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetStats 게시글 통계 조회
func (s *PostService) GetStats() (*models.PostStats, error) {
	stats := &models.PostStats{}

	// 전체 게시글 수
	err := s.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&stats.TotalPosts)
	if err != nil {
		stats.TotalPosts = 0
	}

	// 파일 게시글 수
	err = s.db.QueryRow("SELECT COUNT(*) FROM posts WHERE post_type = 'file'").Scan(&stats.FilePosts)
	if err != nil {
		stats.FilePosts = 0
	}

	// 메시지 게시글 수
	err = s.db.QueryRow("SELECT COUNT(*) FROM posts WHERE post_type = 'message'").Scan(&stats.MessagePosts)
	if err != nil {
		stats.MessagePosts = 0
	}

	return stats, nil
}

// 헬퍼 메서드들

func (s *PostService) scanPost(rows *sql.Rows, includeAdminFields bool) (*models.Post, error) {
	var post models.Post
	var title, content, fileName, filePath sql.NullString
	var fileSize sql.NullInt64
	var ipAddress sql.NullString

	if includeAdminFields {
		err := rows.Scan(&post.ID, &title, &content, &fileName, &filePath, &fileSize, &post.PostType, &ipAddress, &post.CreatedAt, &post.DeletedAt)
		if err != nil {
			return nil, err
		}
		post.IPAddress = ipAddress.String
	} else {
		err := rows.Scan(&post.ID, &title, &content, &fileName, &filePath, &fileSize, &post.PostType, &post.CreatedAt)
		if err != nil {
			return nil, err
		}
	}

	post.Title = title.String
	post.Content = content.String
	post.FileName = fileName.String
	post.FilePath = filePath.String
	post.FileSize = fileSize.Int64
	post.FileSizeMB = float64(fileSize.Int64) / 1048576

	return &post, nil
}

func (s *PostService) getUniqueFileName(directory, originalName string) string {
	ext := filepath.Ext(originalName)
	baseName := strings.TrimSuffix(originalName, ext)
	
	fileName := originalName
	counter := 1
	
	for {
		fullPath := filepath.Join(directory, fileName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		fileName = fmt.Sprintf("%s_%d%s", baseName, counter, ext)
		counter++
	}
	
	return fileName
}

func (s *PostService) moveFileToDeleted(filePath, fileName string, tx *sql.Tx, postID int) error {
	// 삭제된 파일 디렉토리 생성
	if err := os.MkdirAll(s.cfg.File.UploadsDeletedDir, 0755); err != nil {
		return fmt.Errorf("삭제 디렉토리 생성 실패: %v", err)
	}

	// 중복 방지를 위한 고유한 파일명 생성
	timestamp := time.Now().Format("20060102_150405")
	uniqueDeletedFileName := fmt.Sprintf("%d_%s_%s", postID, timestamp, fileName)
	newPath := filepath.Join(s.cfg.File.UploadsDeletedDir, uniqueDeletedFileName)

	// 파일 이동
	if err := os.Rename(filePath, newPath); err != nil {
		return fmt.Errorf("파일 이동 실패: %v", err)
	}

	// 데이터베이스의 파일 경로 업데이트
	_, err := tx.Exec(`
		UPDATE posts 
		SET file_path = $1, file_name = $2 
		WHERE id = $3
	`, newPath, uniqueDeletedFileName, postID)
	
	return err
}
