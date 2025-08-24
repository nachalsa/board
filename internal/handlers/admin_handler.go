package handlers

import (
	"net/http"
	"strconv"

	"file-board/internal/config"
	"file-board/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	postService *services.PostService
	cfg         *config.Config
}

func NewAdminHandler(postService *services.PostService, cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		postService: postService,
		cfg:         cfg,
	}
}

// 관리자 메인 페이지 핸들러
func (h *AdminHandler) IndexHandler(c *gin.Context) {
	posts, err := h.postService.GetAllPosts()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "admin.html", gin.H{"error": "게시글을 불러올 수 없습니다."})
		return
	}

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"posts": posts,
	})
}

// 게시글 삭제 핸들러
func (h *AdminHandler) DeletePostHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 게시글 ID"})
		return
	}

	err = h.postService.DeletePost(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "게시글이 삭제되었습니다."})
}

// 통계 조회 핸들러
func (h *AdminHandler) GetStatsHandler(c *gin.Context) {
	stats, err := h.postService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "통계 조회 실패"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
