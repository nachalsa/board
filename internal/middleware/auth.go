package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	AdminSessionKey = "admin_authenticated"
	SessionMaxAge   = 24 * 60 * 60 // 24시간 (초 단위)
)

// 관리자 인증 미들웨어
func RequireAdminAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		session := sessions.Default(c)
		
		// 세션에서 인증 상태 확인
		if auth := session.Get(AdminSessionKey); auth == nil {
			// 인증되지 않은 경우 로그인 페이지로 리다이렉트
			c.HTML(http.StatusUnauthorized, "admin_login.html", gin.H{})
			c.Abort()
			return
		}
		
		// 세션 만료 시간 갱신
		session.Set(AdminSessionKey, true)
		session.Options(sessions.Options{
			MaxAge:   SessionMaxAge,
			HttpOnly: true,
			Secure:   false, // 개발환경에서는 false, 프로덕션에서는 true
		})
		session.Save()
		
		c.Next()
	})
}

// 관리자 로그인 처리
func HandleAdminLogin(adminPassword string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if c.Request.Method == "GET" {
			// 이미 로그인된 경우 관리자 페이지로 리다이렉트
			session := sessions.Default(c)
			if auth := session.Get(AdminSessionKey); auth != nil {
				c.Redirect(http.StatusFound, "/")
				return
			}
			
			// 로그인 페이지 표시
			c.HTML(http.StatusOK, "admin_login.html", gin.H{})
			return
		}
		
		if c.Request.Method == "POST" {
			password := c.PostForm("password")
			
			if password == "" {
				c.HTML(http.StatusBadRequest, "admin_login.html", gin.H{
					"error": "비밀번호를 입력해주세요.",
				})
				return
			}
			
			if password != adminPassword {
				c.HTML(http.StatusUnauthorized, "admin_login.html", gin.H{
					"error": "비밀번호가 올바르지 않습니다.",
				})
				return
			}
			
			// 로그인 성공 - 세션 설정
			session := sessions.Default(c)
			session.Set(AdminSessionKey, true)
			session.Options(sessions.Options{
				MaxAge:   SessionMaxAge,
				HttpOnly: true,
				Secure:   false, // 개발환경에서는 false, 프로덕션에서는 true
			})
			session.Save()
			
			// 관리자 페이지로 리다이렉트
			c.Redirect(http.StatusFound, "/")
			return
		}
	})
}

// 관리자 로그아웃 처리
func HandleAdminLogout() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 캐시 방지 헤더 설정
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		
		session := sessions.Default(c)
		session.Clear()
		session.Options(sessions.Options{
			MaxAge:   -1, // 즉시 만료
			HttpOnly: true,
			Secure:   false,
		})
		session.Save()
		
		c.Redirect(http.StatusFound, "/login")
	})
}
