package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 관리자 서버용 게시글 구조체
type AdminPost struct {
	ID         int          `json:"id"`
	Title      string       `json:"title"`
	Content    string       `json:"content"`
	FileName   string       `json:"file_name"`
	FilePath   string       `json:"file_path"`
	FileSize   int64        `json:"file_size"`
	FileSizeMB float64      `json:"file_size_mb"`
	PostType   string       `json:"post_type"`
	IPAddress  string       `json:"ip_address"`
	CreatedAt  time.Time    `json:"created_at"`
	DeletedAt  sql.NullTime `json:"deleted_at"`
}

// 로그인 요청 구조체
type AdminLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AdminManager 구조체: 관리자 서버의 모든 상태와 로직을 관리
type AdminManager struct {
	db        *sql.DB
	sessions  map[string]time.Time
	adminUser string
	adminPass string
}

// ------------------- 핵심 변경 사항 -------------------
// 전역 변수 대신 AdminManager 인스턴스를 생성하여 사용하도록 변경합니다.
// 이렇게 하면 main.go의 전역 변수 'db'와 충돌하지 않습니다.
// var adminManager *AdminManager -> 전역 변수 대신 함수 인자로 전달

// 관리자 서버를 시작하는 함수. main.go의 main()에서 호출됩니다.
func startAdminServer() {
	// 1. AdminManager 인스턴스 생성 및 초기화
	adminManager, err := newAdminManager()
	if err != nil {
		log.Fatalf("관리자 매니저 초기화 실패: %v", err)
	}

	// 2. Gin 라우터 설정
	r := gin.Default()

	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// 3. 라우팅 설정
	// 로그인 페이지 및 API
	r.GET("/login", adminManager.loginPageHandler)
	api := r.Group("/api")
	{
		api.POST("/login", adminManager.loginHandler)
		api.POST("/logout", adminManager.logoutHandler)

		// 인증이 필요한 API
		authorizedAPI := api.Use(adminManager.authMiddleware())
		{
			authorizedAPI.DELETE("/delete/:id", adminManager.deletePostHandler)
			authorizedAPI.GET("/stats", adminManager.getPostStatsHandler)
		}
	}

	// 인증이 필요한 페이지
	authorizedPages := r.Use(adminManager.authMiddleware())
	{
		authorizedPages.GET("/", adminManager.adminIndexHandler)
	}

	// 4. 서버 시작
	log.Println("🚀 관리자 서버 시작: http://localhost:8081")
	log.Println("🔐 관리자 로그인: http://localhost:8081/login")
	if err := r.Run(":8081"); err != nil {
		log.Fatal("관리자 서버 시작 실패:", err)
	}
}

// init() 함수는 프로그램 시작 시 자동으로 호출됩니다.
// main.go의 main() 함수보다 먼저 실행될 수 있습니다.
// 여기에 고루틴을 넣어 관리자 서버를 비동기적으로 실행합니다.
func init() {
	go startAdminServer()
}

// newAdminManager: AdminManager를 생성하고 초기화하는 팩토리 함수
func newAdminManager() (*AdminManager, error) {
	// 데이터베이스 연결
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

	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("관리자 데이터베이스 연결 실패: %w", err)
	}

	if err = dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("관리자 데이터베이스 Ping 실패: %w", err)
	}
	log.Println("관리자 데이터베이스 연결 성공")

	// 관리자 계정 정보
	adminUser := os.Getenv("ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASS")
	if adminPass == "" {
		adminPass = "admin123"
		log.Printf("⚠️  관리자 서버 기본 비밀번호 사용 중. ADMIN_PASS 환경변수 설정을 권장합니다!")
	}
	log.Printf("🔐 관리자 사용자: %s", adminUser)

	// AdminManager 인스턴스 생성 및 반환
	return &AdminManager{
		db:        dbConn,
		sessions:  make(map[string]time.Time),
		adminUser: adminUser,
		adminPass: adminPass,
	}, nil
}

// ------------------- 핸들러 및 메서드 (기존과 거의 동일) -------------------
// 이제 모든 핸들러는 AdminManager의 메서드가 됩니다.

func (am *AdminManager) generateSession() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	sessionID := hex.EncodeToString(bytes)
	am.sessions[sessionID] = time.Now().Add(24 * time.Hour)
	return sessionID
}

func (am *AdminManager) validateSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	expiry, exists := am.sessions[sessionID]
	return exists && !time.Now().After(expiry)
}

func (am *AdminManager) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/login" || c.Request.URL.Path == "/api/login" {
			c.Next()
			return
		}
		sessionID, err := c.Cookie("admin_session_id")
		if err != nil || !am.validateSession(sessionID) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (am *AdminManager) loginPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_login.html", nil)
}

func (am *AdminManager) loginHandler(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	if req.Username != am.adminUser || req.Password != am.adminPass {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	sessionID := am.generateSession()
	c.SetCookie("admin_session_id", sessionID, 86400, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

func (am *AdminManager) logoutHandler(c *gin.Context) {
	sessionID, _ := c.Cookie("admin_session_id")
	if sessionID != "" {
		delete(am.sessions, sessionID)
	}
	c.SetCookie("admin_session_id", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

func (am *AdminManager) adminIndexHandler(c *gin.Context) {
	rows, err := am.db.Query(`
		SELECT id, title, content, file_name, file_path, file_size, post_type, ip_address, created_at, deleted_at 
		FROM posts ORDER BY created_at DESC
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

		if err := rows.Scan(&post.ID, &title, &content, &fileName, &filePath, &fileSize, &post.PostType, &ipAddress, &post.CreatedAt, &post.DeletedAt); err != nil {
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
	c.HTML(http.StatusOK, "admin.html", gin.H{"posts": posts})
}

func (am *AdminManager) deletePostHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 게시글 ID"})
		return
	}

	tx, err := am.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "작업 처리 중 오류 발생"})
		return
	}
	defer tx.Rollback()

	var fileName, filePath, postType string
	err = tx.QueryRow(`
		SELECT COALESCE(file_name, ''), COALESCE(file_path, ''), post_type
		FROM posts WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&fileName, &filePath, &postType)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "게시글을 찾을 수 없거나 이미 삭제되었습니다."})
		return
	}

	_, err = tx.Exec("UPDATE posts SET deleted_at = NOW() WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "게시글 삭제 실패"})
		return
	}

	if postType == "file" && filePath != "" {
		if err := os.MkdirAll(UploadsDeletedDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 처리 중 오류 발생"})
			return
		}
		
		uniqueDeletedFileName := getUniqueFileName(UploadsDeletedDir, fileName)
		newPath := filepath.Join(UploadsDeletedDir, uniqueDeletedFileName)

		if err := os.Rename(filePath, newPath); err != nil && !os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 이동에 실패했습니다."})
			return
		}

		_, err = tx.Exec("UPDATE posts SET file_path = $1, file_name = $2 WHERE id = $3", newPath, uniqueDeletedFileName, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "파일 정보 업데이트 실패"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "작업 완료 중 오류 발생"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "게시글이 삭제되었습니다."})
}

func (am *AdminManager) getPostStatsHandler(c *gin.Context) {
	var total, files, messages int
	am.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&total)
	am.db.QueryRow("SELECT COUNT(*) FROM posts WHERE post_type = 'file'").Scan(&files)
	am.db.QueryRow("SELECT COUNT(*) FROM posts WHERE post_type = 'message'").Scan(&messages)

	c.JSON(http.StatusOK, gin.H{"total_posts": total, "file_posts": files, "message_posts": messages})
}
