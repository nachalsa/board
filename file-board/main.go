package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// 게시글 구조체
type Post struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	FileName  string    `json:"file_name"`
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
	PostType  string    `json:"post_type"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

// 데이터베이스 연결
func initDB() {
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

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("데이터베이스 연결 실패:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("데이터베이스 Ping 실패:", err)
	}

	log.Println("데이터베이스 연결 성공")
}

// 파일명 중복 처리
func getUniqueFileName(originalName string) string {
	uploadDir := "./uploads"
	filePath := filepath.Join(uploadDir, originalName)

	// 파일이 존재하지 않으면 원본 이름 반환
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return originalName
	}

	// 파일 확장자와 이름 분리
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)

	// 중복 파일명 처리
	counter := 1
	for {
		newName := fmt.Sprintf("%s(%d)%s", name, counter, ext)
		newPath := filepath.Join(uploadDir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newName
		}
		counter++
	}
}

// 메인 페이지 핸들러
func indexHandler(c *gin.Context) {
	// 게시글 목록 조회
	rows, err := db.Query(`
		SELECT id, title, content, file_name, file_path, file_size, post_type, created_at 
		FROM posts 
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("게시글 조회 실패: %v", err)
		c.HTML(http.StatusInternalServerError, "index.html", gin.H{"error": "게시글을 불러올 수 없습니다."})
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var title, content, fileName, filePath sql.NullString
		var fileSize sql.NullInt64

		err := rows.Scan(&post.ID, &title, &content, &fileName, &filePath, &fileSize, &post.PostType, &post.CreatedAt)
		if err != nil {
			log.Printf("게시글 스캔 실패: %v", err)
			continue
		}

		post.Title = title.String
		post.Content = content.String
		post.FileName = fileName.String
		post.FilePath = filePath.String
		post.FileSize = fileSize.Int64

		posts = append(posts, post)
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"posts": posts,
	})
}

// 파일 업로드 핸들러
func uploadFileHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "파일을 선택해주세요."})
		return
	}
	defer file.Close()

	title := c.PostForm("title")
	if title == "" {
		title = header.Filename
	}

	// 파일 크기 확인 (500MB = 500 * 1024 * 1024)
	if header.Size > 500*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "파일 크기는 500MB를 초과할 수 없습니다."})
		return
	}

	// 업로드 디렉토리 생성
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "업로드 디렉토리 생성 실패"})
		return
	}

	// 중복 파일명 처리
	fileName := getUniqueFileName(header.Filename)
	filePath := filepath.Join(uploadDir, fileName)

	// 파일 저장
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 저장 실패"})
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 복사 실패"})
		return
	}

	// 데이터베이스에 저장
	_, err = db.Exec(`
		INSERT INTO posts (title, file_name, file_path, file_size, post_type) 
		VALUES ($1, $2, $3, $4, $5)
	`, title, fileName, filePath, header.Size, "file")
	if err != nil {
		log.Printf("데이터베이스 저장 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "데이터베이스 저장 실패"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "파일 업로드 성공"})
}

// 메시지 업로드 핸들러
func uploadMessageHandler(c *gin.Context) {
	title := c.PostForm("title")
	content := c.PostForm("content")

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "제목을 입력해주세요."})
		return
	}

	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "내용을 입력해주세요."})
		return
	}

	// 데이터베이스에 저장
	_, err := db.Exec(`
		INSERT INTO posts (title, content, post_type) 
		VALUES ($1, $2, $3)
	`, title, content, "message")
	if err != nil {
		log.Printf("메시지 저장 실패: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "메시지 저장 실패"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "메시지 업로드 성공"})
}

// 파일 다운로드 핸들러
func downloadFileHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 파일 ID"})
		return
	}

	var fileName, filePath string
	err = db.QueryRow(`
		SELECT file_name, file_path 
		FROM posts 
		WHERE id = $1 AND post_type = 'file'
	`, id).Scan(&fileName, &filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "파일을 찾을 수 없습니다."})
		return
	}

	// 파일 존재 확인
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "파일이 존재하지 않습니다."})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.File(filePath)
}

func main() {
	// 데이터베이스 초기화
	initDB()
	defer db.Close()

	// Gin 라우터 설정
	r := gin.Default()

	// 정적 파일 제공
	r.Static("/static", "./static")

	// HTML 템플릿 로드
	r.LoadHTMLGlob("templates/*")

	// 라우팅 설정
	r.GET("/", indexHandler)
	r.POST("/upload/file", uploadFileHandler)
	r.POST("/upload/message", uploadMessageHandler)
	r.GET("/download/:id", downloadFileHandler)

	// 서버 시작
	log.Println("메인 서버 시작: http://localhost:80")
	if err := r.Run(":80"); err != nil {
		log.Fatal("서버 시작 실패:", err)
	}
}
