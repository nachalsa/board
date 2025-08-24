package handlers

import (
	"net/http"
	"os"
	"strconv"

	"file-board/internal/config"
	"file-board/internal/models"
	"file-board/internal/services"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	postService *services.PostService
	cfg         *config.Config
}

func NewHandler(postService *services.PostService, cfg *config.Config) *Handler {
	return &Handler{
		postService: postService,
		cfg:         cfg,
	}
}

// 메인 페이지 핸들러
func (h *Handler) IndexHandler(c *gin.Context) {
	posts, err := h.postService.GetPosts()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "index.html", gin.H{
			"error": "게시글을 불러올 수 없습니다.",
			"maxFileSize": h.cfg.GetMaxFileSizeText(),
		})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"posts": posts,
		"maxFileSize": h.cfg.GetMaxFileSizeText(),
	})
}

// 파일 업로드 핸들러
func (h *Handler) UploadFileHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "파일을 선택해주세요."})
		return
	}
	defer file.Close()

	title := c.PostForm("title")
	ipAddress := c.ClientIP()

	err = h.postService.CreateFilePost(title, header, ipAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "파일 업로드 성공"})
}

// 메시지 업로드 핸들러
func (h *Handler) UploadMessageHandler(c *gin.Context) {
	// JSON과 Form-data 모두 처리
	var title, content string
	
	// Content-Type 확인
	contentType := c.GetHeader("Content-Type")
	if contentType == "application/json" {
		// JSON 요청 처리 (API)
		var req models.MessageUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "제목과 내용을 입력해주세요."})
			return
		}
		title = req.Title
		content = req.Content
	} else {
		// Form-data 요청 처리 (웹 폼)
		title = c.PostForm("title")
		content = c.PostForm("content")
	}

	// 제목은 필수, 내용은 선택사항
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "제목을 입력해주세요."})
		return
	}

	ipAddress := c.ClientIP()

	err := h.postService.CreateMessagePost(title, content, ipAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "메시지 업로드 실패"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "메시지 업로드 성공"})
}

// 파일 다운로드 핸들러
func (h *Handler) DownloadFileHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 파일 ID"})
		return
	}

	fileName, filePath, err := h.postService.GetPostFile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "파일을 찾을 수 없습니다."})
		return
	}

	// 파일 존재 확인
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "파일이 존재하지 않습니다."})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.File(filePath)
}
