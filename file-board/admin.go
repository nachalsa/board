package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
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
	PostType  string    `json:"post_type"`
	CreatedAt time.Time `json:"created_at"`
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
		SELECT id, title, content, file_name, file_path, file_size, post_type, created_at 
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
	for rows.Next() {
		var post AdminPost
		var title, content, fileName, filePath sql.NullString
		var fileSize sql.NullInt64

		err := rows.Scan(&post.ID, &title, &content, &fileName, &filePath, &fileSize, &post.PostType, &post.CreatedAt)
		if err != nil {
			log.Printf("관리자 게시글 스캔 실패: %v", err)
			continue
		}

		post.Title = title.String
		post.Content = content.String
		post.FileName = fileName.String
		post.FilePath = filePath.String
		post.FileSize = fileSize.Int64

		posts = append(posts, post)
	}

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"posts": posts,
	})
}

// 게시글 삭제 핸들러
func deletePostHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 게시글 ID"})
		return
	}

	// 게시글 정보 조회
	var fileName, filePath, postType string
	err = adminDB.QueryRow(`
		SELECT COALESCE(file_name, ''), COALESCE(file_path, ''), post_type 
		FROM posts 
		WHERE id = $1
	`, id).Scan(&fileName, &filePath, &postType)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "게시글을 찾을 수 없습니다."})
			return
		}
		log.Printf("게시글 정보 조회 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "게시글 정보 조회 실패"})
		return
	}

	// 데이터베이스에서 게시글 삭제
	_, err = adminDB.Exec("DELETE FROM posts WHERE id = $1", id)
	if err != nil {
		log.Printf("게시글 삭제 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "게시글 삭제 실패"})
		return
	}

	// 파일 타입인 경우 물리적 파일도 삭제
	if postType == "file" && filePath != "" {
		if err := os.Remove(filePath); err != nil {
			log.Printf("파일 삭제 실패: %v", err)
			// 파일 삭제 실패는 치명적이지 않으므로 계속 진행
		}
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
