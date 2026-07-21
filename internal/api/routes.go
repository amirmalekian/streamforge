package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"streamforge/internal/auth"
	"streamforge/internal/downloader"
	"streamforge/internal/jobs"
	"streamforge/internal/middleware"
	"streamforge/internal/queue"
	"streamforge/internal/redis"
)

type jobCreateService interface {
	CreateJob(ctx context.Context, userID, sourceURL string) (*jobs.JobResponse, error)
	CreateMediaItems(ctx context.Context, jobID string, items []jobs.MediaItemInput) error
	UpdateJobStatus(ctx context.Context, jobID, status, errorMsg string) error
}

type queuePublisher interface {
	Publish(ctx context.Context, msg queue.Message) error
}

func RegisterRoutes(
	r *gin.Engine,
	authSvc *auth.Service,
	jobSvc *jobs.Service,
	queueSvc *queue.Service,
	redisClient *redis.Client,
	dl downloader.Downloader,
) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/register", registerHandler(authSvc))
		authGroup.POST("/login", loginHandler(authSvc))
	}

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(authSvc))
	{
		jobsGroup := api.Group("/jobs")
		{
			jobsGroup.POST("", createJobHandler(jobSvc, queueSvc, dl))
			jobsGroup.GET("", listJobsHandler(jobSvc))
			jobsGroup.GET("/:id", getJobHandler(jobSvc))
			jobsGroup.DELETE("/:id", cancelJobHandler(jobSvc))
			jobsGroup.GET("/:id/events", sseHandler(jobSvc, redisClient))
			jobsGroup.GET("/:id/items", getMediaItemsHandler(jobSvc))
		}
	}
}

func registerHandler(svc *auth.Service) gin.HandlerFunc {
	type request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}

	return func(c *gin.Context) {
		var req request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
			return
		}

		resp, err := svc.Register(c.Request.Context(), auth.RegisterRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if err == auth.ErrUserExists {
				c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

func loginHandler(svc *auth.Service) gin.HandlerFunc {
	type request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	return func(c *gin.Context) {
		var req request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
			return
		}

		resp, err := svc.Login(c.Request.Context(), auth.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if err == auth.ErrInvalidCredentials {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func createJobHandler(svc jobCreateService, queueSvc queuePublisher, dl downloader.Downloader) gin.HandlerFunc {
	type request struct {
		SourceURL string `json:"source_url" binding:"required,url"`
	}

	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
			return
		}

		resp, err := svc.CreateJob(c.Request.Context(), userIDStr, req.SourceURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
			return
		}

		// Check if the URL is a playlist
		var playlist *downloader.Playlist
		var playlistErr error
		if dl != nil {
			playlist, playlistErr = dl.GetPlaylist(c.Request.Context(), req.SourceURL)
		}
		if playlistErr != nil && !errors.Is(playlistErr, downloader.ErrNotAPlaylist) {
			_ = svc.UpdateJobStatus(c.Request.Context(), resp.ID.String(), "FAILED", "failed to fetch playlist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch playlist"})
			return
		}

		if playlistErr != nil || playlist == nil || len(playlist.Entries) == 0 {
			// Not a playlist or failed to fetch, treat as single video
			if err := svc.CreateMediaItems(c.Request.Context(), resp.ID.String(), []jobs.MediaItemInput{
				{Title: "Source Media", SourceURL: req.SourceURL},
			}); err != nil {
				_ = svc.UpdateJobStatus(c.Request.Context(), resp.ID.String(), "FAILED", "failed to create media items")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create media items"})
				return
			}
		} else {
			// It's a playlist, create media items for each entry
			items := make([]jobs.MediaItemInput, len(playlist.Entries))
			for i, entry := range playlist.Entries {
				items[i] = jobs.MediaItemInput{
					Title:     entry.Title,
					SourceURL: entry.URL,
				}
			}
			if err := svc.CreateMediaItems(c.Request.Context(), resp.ID.String(), items); err != nil {
				_ = svc.UpdateJobStatus(c.Request.Context(), resp.ID.String(), "FAILED", "failed to create media items")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create media items"})
				return
			}
		}

		if queueSvc == nil {
			_ = svc.UpdateJobStatus(c.Request.Context(), resp.ID.String(), "FAILED", "queue service unavailable")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "queue service unavailable"})
			return
		}

		if err := queueSvc.Publish(c.Request.Context(), queue.Message{
			JobID:     resp.ID.String(),
			Action:    "PROCESS",
			Payload:   map[string]interface{}{"source_url": req.SourceURL},
			CreatedAt: time.Now().Format(time.RFC3339),
		}); err != nil {
			_ = svc.UpdateJobStatus(c.Request.Context(), resp.ID.String(), "FAILED", "failed to publish queue message")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue job", "details": err.Error()})
			return
		}

		_ = svc.UpdateJobStatus(c.Request.Context(), resp.ID.String(), "QUEUED", "")
		c.JSON(http.StatusCreated, resp)
	}
}

func listJobsHandler(svc *jobs.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		page := 1
		if p := c.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil {
				page = v
			}
		}

		pageSize := 20
		if ps := c.Query("page_size"); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil {
				pageSize = v
			}
		}

		status := c.Query("status")

		resp, total, err := svc.ListJobs(c.Request.Context(), userIDStr, status, pageSize, (page-1)*pageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"jobs": resp,
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + pageSize - 1) / pageSize,
			},
		})
	}
}

func getJobHandler(svc *jobs.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		jobID := c.Param("id")

		resp, err := svc.GetJob(c.Request.Context(), userIDStr, jobID)
		if err != nil {
			if err == jobs.ErrJobNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
				return
			}
			if err == jobs.ErrUnauthorized {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get job"})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func cancelJobHandler(svc *jobs.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		jobID := c.Param("id")

		if err := svc.CancelJob(c.Request.Context(), userIDStr, jobID); err != nil {
			if err == jobs.ErrJobNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
				return
			}
			if err == jobs.ErrUnauthorized {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			if err == jobs.ErrCannotCancel {
				c.JSON(http.StatusConflict, gin.H{"error": "cannot cancel", "message": "job is in terminal state"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel job"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "job cancelled"})
	}
}

func sseHandler(svc *jobs.Service, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		jobID := c.Param("id")

		job, err := svc.GetJob(c.Request.Context(), userIDStr, jobID)
		if err != nil {
			if err == jobs.ErrJobNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
				return
			}
			if err == jobs.ErrUnauthorized {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get job"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		clientChan := make(chan string)
		defer close(clientChan)

		go func() {
			for {
				select {
				case <-c.Request.Context().Done():
					return
				case msg := <-clientChan:
					c.SSEvent("message", msg)
					c.Writer.Flush()
				}
			}
		}()

		svc.Subscribe(c.Request.Context(), jobID, clientChan)

		// Send initial connection event
		c.SSEvent("connected", gin.H{"job_id": jobID, "status": job.Status})
		c.Writer.Flush()

		// Keep connection alive
		<-c.Request.Context().Done()
	}
}

func getMediaItemsHandler(svc *jobs.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		jobID := c.Param("id")

		page := 1
		if p := c.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil {
				page = v
			}
		}

		pageSize := 20
		if ps := c.Query("page_size"); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil {
				pageSize = v
			}
		}

		status := c.Query("status")

		resp, total, err := svc.GetMediaItems(c.Request.Context(), userIDStr, jobID, status, pageSize, (page-1)*pageSize)
		if err != nil {
			if err == jobs.ErrJobNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
				return
			}
			if err == jobs.ErrUnauthorized {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get media items"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": resp,
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + pageSize - 1) / pageSize,
			},
		})
	}
}
