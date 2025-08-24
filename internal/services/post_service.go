package services

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"

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

// GetPosts 게시글 목록 조회 (일반 사용자용 - 삭제된 것 제외)
func (s *PostService) GetPosts() ([]models.Post, error) {
	return s.getPosts(false)
}

// GetAllPosts 모든 게시글 조회 (관리자용 - 삭제된 것 포함)
func (s *PostService) GetAllPosts() ([]models.Post, error) {
	return s.getPosts(true)
}

// getPosts 통합 게시글 조회 메서드 (files 테이블 조인)
func (s *PostService) getPosts(includeDeleted bool) ([]models.Post, error) {
	whereClause := ""
	if !includeDeleted {
		whereClause = "WHERE p.deleted_at IS NULL"
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.title, COALESCE(p.content, '') as content, p.file_name, p.file_id, 
		       p.post_type, p.created_at%s,
		       f.id, f.file_hash, f.file_path, f.file_size, f.mime_type
		FROM posts p
		LEFT JOIN files f ON p.file_id = f.id
		%s
		ORDER BY p.created_at DESC
	`, func() string {
		if includeDeleted {
			return ", p.deleted_at"
		}
		return ""
	}(), whereClause)

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("게시글 조회 실패: %v", err)
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		post, err := s.scanPost(rows, includeDeleted)
		if err != nil {
			fmt.Printf("게시글 스캔 에러 (ID 건너뜀): %v\n", err)
			continue
		}
		posts = append(posts, *post)
	}

	return posts, nil
}

// scanPost 게시글 스캔 헬퍼 (files 테이블 조인 지원)
func (s *PostService) scanPost(rows *sql.Rows, includeDeleted bool) (*models.Post, error) {
	var post models.Post
	var fileName sql.NullString
	var fileID sql.NullInt32
	
	// files 테이블 필드들
	var fID sql.NullInt32
	var fHash, fPath, fMimeType sql.NullString
	var fSize sql.NullInt64

	if includeDeleted {
		err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &fileName, &fileID,
			&post.PostType, &post.CreatedAt, &post.DeletedAt,
			&fID, &fHash, &fPath, &fSize, &fMimeType,
		)
		if err != nil {
			return nil, err
		}
	} else {
		err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &fileName, &fileID,
			&post.PostType, &post.CreatedAt,
			&fID, &fHash, &fPath, &fSize, &fMimeType,
		)
		if err != nil {
			return nil, err
		}
	}

	// 파일명 설정
	if fileName.Valid {
		post.FileName = fileName.String
	}
	
	// file_id 설정
	post.FileID = fileID

	// 파일 정보 설정 (files 테이블에서 조인된 데이터)
	if fID.Valid && fPath.Valid {
		post.File = &models.File{
			ID:       int(fID.Int32),
			FileHash: fHash.String,
			FilePath: fPath.String,
			FileSize: fSize.Int64,
			MimeType: fMimeType.String,
		}
	}

	return &post, nil
}

// GetFileInfo 파일 정보 조회 (files 테이블 조인)
func (s *PostService) GetFileInfo(id int) (string, string, error) {
	var fileName, filePath string
	err := s.db.QueryRow(`
		SELECT p.file_name, f.file_path 
		FROM posts p 
		JOIN files f ON p.file_id = f.id 
		WHERE p.id = $1 AND p.post_type = 'file'
	`, id).Scan(&fileName, &filePath)
	
	return fileName, filePath, err
}

// CreateFilePost 파일 게시글 생성 - files 테이블 활용
func (s *PostService) CreateFilePost(title string, file *multipart.FileHeader, ipAddress string) error {
	if err := s.validateFile(file); err != nil {
		return err
	}

	// 디렉토리 생성
	if err := os.MkdirAll(s.cfg.File.UploadsDir, 0755); err != nil {
		return fmt.Errorf("업로드 디렉토리 생성 실패: %v", err)
	}

	// 파일 해시 생성
	fileHash, err := s.generateFileHash(file)
	if err != nil {
		return err
	}

	// files 테이블에서 중복 파일 확인
	var fileID int
	err = s.db.QueryRow(`
		SELECT id FROM files WHERE file_hash = $1
	`, fileHash).Scan(&fileID)

	if err == sql.ErrNoRows {
		// 새 파일 저장
		filePath := filepath.Join(s.cfg.File.UploadsDir, fileHash)
		
		if err := s.saveFile(file, filePath); err != nil {
			return err
		}

		// files 테이블에 저장
		err = s.db.QueryRow(`
			INSERT INTO files (file_hash, file_path, file_size, mime_type) 
			VALUES ($1, $2, $3, $4) RETURNING id
		`, fileHash, filePath, file.Size, file.Header.Get("Content-Type")).Scan(&fileID)
		
		if err != nil {
			return fmt.Errorf("파일 정보 저장 실패: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("중복 파일 확인 실패: %v", err)
	}
	// else: 기존 파일 ID 사용 (중복 파일)

	// 제목 설정
	if title == "" {
		title = file.Filename
	}

	// posts 테이블에 저장 (file_id 참조)
	return s.savePostToDb(title, file.Filename, fileID, "file", ipAddress)
}

// CreateMessagePost 메시지 게시글 생성
func (s *PostService) CreateMessagePost(title, content, ipAddress string) error {
	return s.savePostToDb(title, "", 0, "message", ipAddress, content)
}

// DeletePost 게시글 삭제 (소프트 삭제)
func (s *PostService) DeletePost(id int) error {
	return s.updatePostStatus(id, "SET deleted_at = NOW()", "삭제", "deleted_at IS NULL")
}

// RestorePost 게시글 복구
func (s *PostService) RestorePost(id int) error {
	return s.updatePostStatus(id, "SET deleted_at = NULL", "복구", "deleted_at IS NOT NULL")
}

// GetStats 통계 조회 (files 테이블 포함)
func (s *PostService) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	queries := map[string]string{
		"total_posts":   "SELECT COUNT(*) FROM posts",
		"active_posts":  "SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL",
		"deleted_posts": "SELECT COUNT(*) FROM posts WHERE deleted_at IS NOT NULL",
		"file_posts":    "SELECT COUNT(*) FROM posts WHERE post_type = 'file'",
		"message_posts": "SELECT COUNT(*) FROM posts WHERE post_type = 'message'",
		"unique_files":  "SELECT COUNT(*) FROM files", // 고유 파일 수
	}

	for key, query := range queries {
		var count int
		if err := s.db.QueryRow(query).Scan(&count); err != nil {
			return nil, fmt.Errorf("%s 조회 실패: %v", key, err)
		}
		stats[key] = count
	}

	// 총 파일 크기 계산 (files 테이블에서)
	var totalSize sql.NullInt64
	err := s.db.QueryRow("SELECT SUM(file_size) FROM files").Scan(&totalSize)
	if err != nil {
		return nil, fmt.Errorf("파일 크기 합계 조회 실패: %v", err)
	}
	
	if totalSize.Valid {
		stats["total_storage_bytes"] = totalSize.Int64
		stats["total_storage_mb"] = float64(totalSize.Int64) / (1024 * 1024)
	} else {
		stats["total_storage_bytes"] = 0
		stats["total_storage_mb"] = 0.0
	}

	return stats, nil
}

// === 헬퍼 메서드들 ===

// validateFile 파일 유효성 검사
func (s *PostService) validateFile(file *multipart.FileHeader) error {
	if file.Size > s.cfg.File.MaxFileSize {
		return fmt.Errorf("파일 크기는 %s를 초과할 수 없습니다", s.cfg.GetMaxFileSizeText())
	}
	return nil
}

// generateFileHash FNV 해시 생성 (파일 내용 기반 - 중복 파일 감지용)
func (s *PostService) generateFileHash(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("파일 열기 실패: %v", err)
	}
	defer src.Close()

	hasher := fnv.New64a()
	if _, err := io.Copy(hasher, src); err != nil {
		return "", fmt.Errorf("파일 해시 생성 실패: %v", err)
	}

	return strconv.FormatUint(hasher.Sum64(), 16), nil
}

// saveFile 파일을 디스크에 저장
func (s *PostService) saveFile(file *multipart.FileHeader, filePath string) error {
	// 중복 파일 확인
	if _, err := os.Stat(filePath); err == nil {
		// 파일이 이미 존재하면 저장하지 않음 (중복 제거)
		return nil
	}

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

	return nil
}

// savePostToDb 게시글을 데이터베이스에 저장 (files 테이블 구조용)
func (s *PostService) savePostToDb(title, fileName string, fileID int, postType, ipAddress string, content ...string) error {
	var contentStr string
	if len(content) > 0 {
		contentStr = content[0]
	}

	var query string
	var args []interface{}

	if postType == "file" {
		query = `
			INSERT INTO posts (title, content, file_name, file_id, post_type, ip_address) 
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		args = []interface{}{title, contentStr, fileName, fileID, postType, ipAddress}
	} else {
		query = `
			INSERT INTO posts (title, content, post_type, ip_address) 
			VALUES ($1, $2, $3, $4)
		`
		args = []interface{}{title, contentStr, postType, ipAddress}
	}

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("데이터베이스 저장 실패: %v", err)
	}
	return nil
}

// updatePostStatus 게시글 상태 업데이트 (삭제/복구 통합)
func (s *PostService) updatePostStatus(id int, setClause, action, whereCondition string) error {
	query := fmt.Sprintf("UPDATE posts %s WHERE id = $1 AND %s", setClause, whereCondition)
	
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("게시글 %s 실패: %v", action, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s 결과 확인 실패: %v", action, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("게시글을 찾을 수 없거나 이미 %s되었습니다", action)
	}

	return nil
}
