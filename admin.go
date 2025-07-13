package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// 관리자 서버용 게시글 구조체
type AdminPost struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	FileName  string    `json:"file_name"`
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
	FileSizeMB	float64 `json:"file_size_mb"`
	PostType  string    `json:"post_type"`
	IPAddress	string	`json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
	DeletedAt sql.NullTime `json:"deleted_at"`
}

var adminDB *sql.DB

// 관리자 데이터베이스 연결
func initAdminDB() {
	var err error
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	adminDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("관리자 데이터베이스 연결 실패:", err)
	}

	if err = adminDB.Ping(); err != nil {
		log.Fatal("관리자 데이터베이스 Ping 실패:", err)
	}

	log.Println("관리자 데이터베이스 연결 성공")
}

// 관리자 페이지 핸들러
func adminIndexHandler(c *gin.Context) {
	// 게시글 목록 조회
	rows, err := adminDB.Query(`
		SELECT id, title, content, file_name, file_path, file_size, post_type, ip_address, created_at, deleted_at 
		FROM posts 
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("관리자 게시글 조회 실패: %v", err)
		c.HTML(http.StatusInternalServerError, "admin.html", gin.H{"error": "게시글을 불러올 수 없습니다."})
		return
	}
	defer rows.Close()

	var posts []AdminPost
	kst, err := time.LoadLocation("Asia/Seoul")
    if err != nil {
        log.Printf("관리자 페이지 시간대 로드 실패: %v", err)
        kst = time.UTC
    }
	for rows.Next() {
		var post AdminPost
		var title, content, fileName, filePath, ipAddress sql.NullString
		var fileSize sql.NullInt64

		err := rows.Scan(&post.ID, &title, &content, &fileName, &filePath, &fileSize, &post.PostType, &ipAddress, &post.CreatedAt, &post.DeletedAt)
		if err != nil {
			log.Printf("관리자 게시글 스캔 실패: %v", err)
			continue
		}

		post.Title = title.String
		post.Content = content.String
		post.FileName = fileName.String
		post.FilePath = filePath.String
		post.FileSize = fileSize.Int64
		post.FileSizeMB = float64(fileSize.Int64) / 1048576
		post.IPAddress = ipAddress.String 
		post.CreatedAt = post.CreatedAt.In(kst)

		
		posts = append(posts, post)
	}

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"posts": posts,
	})
}

// 게시글 삭제 핸들러
// admin.go

func deletePostHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 게시글 ID"})
		return
	}

	tx, err := adminDB.Begin()
	if err != nil {
		log.Printf("트랜잭션 시작 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "작업 처리 중 오류 발생"})
		return
	}
	defer tx.Rollback()

	var fileName, filePath, postType string
	err = tx.QueryRow(`
		SELECT COALESCE(file_name, ''), COALESCE(file_path, ''), post_type
		FROM posts
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&fileName, &filePath, &postType) // 삭제되지 않은 게시물만 대상으로 변경

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "게시글을 찾을 수 없거나 이미 삭제되었습니다."})
			return
		}
		log.Printf("게시글 정보 조회 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "게시글 정보 조회 실패"})
		return
	}

	// 1. 데이터베이스에서 soft delete 처리 (deleted_at 설정)
	_, err = tx.Exec("UPDATE posts SET deleted_at = NOW() WHERE id = $1", id)
	if err != nil {
		log.Printf("게시글 삭제(soft delete) 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "게시글 삭제 실패"})
		return
	}

	// 파일 타입인 경우에만 파일 이동 및 DB 추가 업데이트
	if postType == "file" && filePath != "" {
		// --- 디버깅 로그 시작 ---
		log.Printf("----- 파일 삭제 처리 시작 (Post ID: %d) -----", id)
		log.Printf("원본 파일명 (DB): %s", fileName)
		log.Printf("원본 파일 경로 (DB): %s", filePath)

		deletedDir := "./uploads_deleted"
		if err := os.MkdirAll(deletedDir, 0755); err != nil {
			log.Printf("삭제된 파일 보관 디렉토리 생성 실패: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 처리 중 오류 발생"})
			return
		}

		uniqueDeletedFileName := getUniqueFileName(deletedDir, fileName)
		newPath := filepath.Join(deletedDir, uniqueDeletedFileName)

		log.Printf("생성된 고유 파일명: %s", uniqueDeletedFileName)
		log.Printf("새 파일 경로: %s", newPath)
		log.Printf("실행할 명령: os.Rename(\"%s\", \"%s\")", filePath, newPath)
		// --- 디버깅 로그 끝 ---

		// 2. 파일을 새 경로로 이동
		if err := os.Rename(filePath, newPath); err != nil {
			// 파일이 이미 없는 경우는 무시
			if !os.IsNotExist(err) {
				log.Printf("파일 이동 실패: %v", err) // 실제 에러 로그 출력
				c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 이동에 실패했습니다."})
				return
			}
			log.Printf("파일이 원본 경로에 존재하지 않아 이동을 건너뜁니다: %s", filePath)
		}

		// 3. (★중요★) 파일 이동 후, DB의 경로와 파일명도 업데이트하여 데이터 일관성 유지
		_, err = tx.Exec(`
			UPDATE posts
			SET file_path = $1, file_name = $2
			WHERE id = $3
		`, newPath, uniqueDeletedFileName, id)
		if err != nil {
			log.Printf("삭제 후 파일 경로 업데이트 실패: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 정보 업데이트 실패"})
			return
		}
		log.Printf("DB 파일 정보 업데이트 완료: path=%s, name=%s", newPath, uniqueDeletedFileName)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("트랜잭션 커밋 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "작업 완료 중 오류 발생"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "게시글이 삭제되었습니다."})
}

// 전체 게시글 개수 조회
func getPostStatsHandler(c *gin.Context) {
	var totalPosts, filePosts, messagePosts int

	// 전체 게시글 수
	err := adminDB.QueryRow("SELECT COUNT(*) FROM posts").Scan(&totalPosts)
	if err != nil {
		log.Printf("전체 게시글 수 조회 실패: %v", err)
		totalPosts = 0
	}

	// 파일 게시글 수
	err = adminDB.QueryRow("SELECT COUNT(*) FROM posts WHERE post_type = 'file'").Scan(&filePosts)
	if err != nil {
		log.Printf("파일 게시글 수 조회 실패: %v", err)
		filePosts = 0
	}

	// 메시지 게시글 수
	err = adminDB.QueryRow("SELECT COUNT(*) FROM posts WHERE post_type = 'message'").Scan(&messagePosts)
	if err != nil {
		log.Printf("메시지 게시글 수 조회 실패: %v", err)
		messagePosts = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"total_posts":   totalPosts,
		"file_posts":    filePosts,
		"message_posts": messagePosts,
	})
}

// 관리자 서버 시작
func startAdminServer() {
	// 관리자 데이터베이스 초기화
	initAdminDB()

	// Gin 라우터 설정
	r := gin.Default()

	// 정적 파일 제공
	r.Static("/static", "./static")

	// HTML 템플릿 로드
	r.LoadHTMLGlob("templates/*")

	// 라우팅 설정
	r.GET("/", adminIndexHandler)
	r.DELETE("/delete/:id", deletePostHandler)
	r.GET("/stats", getPostStatsHandler)

	// 서버 시작
	log.Println("관리자 서버 시작: http://localhost:8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal("관리자 서버 시작 실패:", err)
	}
}

func init() {
	// 관리자 서버를 고루틴으로 실행
	go startAdminServer()
}
