package main

import (
	"html/template"
	"log"
	"time"

	"file-board/internal/config"
	"file-board/internal/database"
	"file-board/internal/handlers"
	"file-board/internal/middleware"
	"file-board/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	// 설정 로드
	cfg := config.Load()

	// 데이터베이스 연결
	db, err := database.New(cfg)
	if err != nil {
		log.Fatal("데이터베이스 연결 실패:", err)
	}
	defer db.Close()

	// 서비스 초기화
	postService := services.NewPostService(db.GetConnection(), cfg)

	// 핸들러 초기화
	userHandler := handlers.NewHandler(postService, cfg)
	adminHandler := handlers.NewAdminHandler(postService, cfg)

	// 사용자 서버 시작 (고루틴)
	go startUserServer(userHandler, cfg)

	// 관리자 서버 시작
	startAdminServer(adminHandler, userHandler, cfg)
}

func startUserServer(handler *handlers.Handler, cfg *config.Config) {
	r := gin.Default()

	// 정적 파일 제공
	r.Static("/static", "./web/static")

	// 템플릿 함수 등록
	r.SetFuncMap(template.FuncMap{
		"kstTime": func(t time.Time) string {
			loc, _ := time.LoadLocation("Asia/Seoul")
			return t.In(loc).Format("2006-01-02 15:04")
		},
		"kstTimeISO": func(t time.Time) string {
			loc, _ := time.LoadLocation("Asia/Seoul")
			return t.In(loc).Format("2006-01-02T15:04:05Z07:00")
		},
	})

	// HTML 템플릿 로드
	r.LoadHTMLGlob("web/templates/*")

	// 라우팅 설정
	r.GET("/", handler.IndexHandler)
	r.POST("/upload/file", handler.UploadFileHandler)
	r.POST("/upload/message", handler.UploadMessageHandler)
	r.GET("/download/:id", handler.DownloadFileHandler)

	log.Printf("사용자 서버 시작: http://localhost:%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("사용자 서버 시작 실패:", err)
	}
}

func startAdminServer(adminHandler *handlers.AdminHandler, userHandler *handlers.Handler, cfg *config.Config) {
	r := gin.Default()

	// 세션 설정
	store := cookie.NewStore([]byte("secret-admin-session-key-change-this-in-production"))
	r.Use(sessions.Sessions("admin-session", store))

	// 정적 파일 제공
	r.Static("/static", "./web/static")

	// 템플릿 함수 등록 (관리자도 동일)
	r.SetFuncMap(template.FuncMap{
		"kstTime": func(t time.Time) string {
			loc, _ := time.LoadLocation("Asia/Seoul")
			return t.In(loc).Format("2006-01-02 15:04")
		},
		"kstTimeISO": func(t time.Time) string {
			loc, _ := time.LoadLocation("Asia/Seoul")
			return t.In(loc).Format("2006-01-02T15:04:05Z07:00")
		},
	})

	// HTML 템플릿 로드
	r.LoadHTMLGlob("web/templates/*")

	// 로그인/로그아웃 라우팅 (인증 미들웨어 적용 전)
	r.Any("/login", middleware.HandleAdminLogin(cfg.Server.AdminPassword))
	r.GET("/logout", middleware.HandleAdminLogout())

	// 관리자 인증이 필요한 라우팅
	adminGroup := r.Group("/")
	adminGroup.Use(middleware.RequireAdminAuth())
	{
		adminGroup.GET("/", adminHandler.IndexHandler)
		adminGroup.DELETE("/delete/:id", adminHandler.DeletePostHandler)
		adminGroup.POST("/restore/:id", adminHandler.RestorePostHandler)
		adminGroup.GET("/stats", adminHandler.GetStatsHandler)
		adminGroup.GET("/download/:id", userHandler.DownloadFileHandler) // 동일한 다운로드 핸들러 재사용
	}

	log.Printf("관리자 서버 시작: http://localhost:%s", cfg.Server.AdminPort)
	if err := r.Run(":" + cfg.Server.AdminPort); err != nil {
		log.Fatal("관리자 서버 시작 실패:", err)
	}
}
