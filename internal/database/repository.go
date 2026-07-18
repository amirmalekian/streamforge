package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *Repository) Close() {
	r.pool.Close()
}

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    string
	UpdatedAt    string
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	user := &User{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

type Job struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	SourceURL      string
	Status         string
	TotalItems     int
	CompletedItems int
	ErrorMessage   string
	CreatedAt      string
	UpdatedAt      string
}

func (r *Repository) CreateJob(ctx context.Context, userID uuid.UUID, sourceURL string) (*Job, error) {
	job := &Job{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO jobs (user_id, source_url)
		VALUES ($1, $2)
		RETURNING id, user_id, source_url, status, total_items, completed_items, error_message, created_at, updated_at
	`, userID, sourceURL).Scan(
		&job.ID, &job.UserID, &job.SourceURL, &job.Status,
		&job.TotalItems, &job.CompletedItems, &job.ErrorMessage,
		&job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *Repository) GetJob(ctx context.Context, jobID string) (*Job, error) {
	job := &Job{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, source_url, status, total_items, completed_items, error_message, created_at, updated_at
		FROM jobs WHERE id = $1
	`, jobID).Scan(
		&job.ID, &job.UserID, &job.SourceURL, &job.Status,
		&job.TotalItems, &job.CompletedItems, &job.ErrorMessage,
		&job.CreatedAt, &job.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *Repository) ListJobs(ctx context.Context, userID string, status string, limit, offset int) ([]*Job, int, error) {
	query := `
		SELECT id, user_id, source_url, status, total_items, completed_items, error_message, created_at, updated_at
		FROM jobs WHERE user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount+1, argCount+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		if err := rows.Scan(
			&job.ID, &job.UserID, &job.SourceURL, &job.Status,
			&job.TotalItems, &job.CompletedItems, &job.ErrorMessage,
			&job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM jobs WHERE user_id = $1`
	countArgs := []interface{}{userID}
	if status != "" {
		countQuery += ` AND status = $2`
		countArgs = append(countArgs, status)
	}
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (r *Repository) UpdateJobStatus(ctx context.Context, jobID, status, errorMsg string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3
	`, status, errorMsg, jobID)
	return err
}

func (r *Repository) UpdateJobProgress(ctx context.Context, jobID string, completed int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs SET completed_items = $1, updated_at = NOW()
		WHERE id = $2
	`, completed, jobID)
	return err
}

func (r *Repository) SetJobTotalItems(ctx context.Context, jobID string, total int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs SET total_items = $1, updated_at = NOW()
		WHERE id = $2
	`, total, jobID)
	return err
}

type MediaItem struct {
	ID           uuid.UUID
	JobID        uuid.UUID
	Title        string
	SourceURL    string
	Status       string
	Progress     int
	SizeBytes    int64
	ErrorMessage string
	CreatedAt    string
	UpdatedAt    string
}

type CreateMediaItemParams struct {
	JobID     uuid.UUID
	Title     string
	SourceURL string
}

func (r *Repository) CreateMediaItem(ctx context.Context, params CreateMediaItemParams) (*MediaItem, error) {
	item := &MediaItem{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO media_items (job_id, title, source_url)
		VALUES ($1, $2, $3)
		RETURNING id, job_id, title, source_url, status, progress, size_bytes, error_message, created_at, updated_at
	`, params.JobID, params.Title, params.SourceURL).Scan(
		&item.ID, &item.JobID, &item.Title, &item.SourceURL,
		&item.Status, &item.Progress, &item.SizeBytes, &item.ErrorMessage,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) GetMediaItem(ctx context.Context, itemID string) (*MediaItem, error) {
	item := &MediaItem{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, job_id, title, source_url, status, progress, size_bytes, error_message, created_at, updated_at
		FROM media_items WHERE id = $1
	`, itemID).Scan(
		&item.ID, &item.JobID, &item.Title, &item.SourceURL,
		&item.Status, &item.Progress, &item.SizeBytes, &item.ErrorMessage,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) ListMediaItems(ctx context.Context, jobID string, status string, limit, offset int) ([]*MediaItem, int, error) {
	query := `
		SELECT id, job_id, title, source_url, status, progress, size_bytes, error_message, created_at, updated_at
		FROM media_items WHERE job_id = $1
	`
	args := []interface{}{jobID}
	argCount := 1

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	query += fmt.Sprintf(" ORDER BY created_at ASC LIMIT $%d OFFSET $%d", argCount+1, argCount+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*MediaItem
	for rows.Next() {
		item := &MediaItem{}
		if err := rows.Scan(
			&item.ID, &item.JobID, &item.Title, &item.SourceURL,
			&item.Status, &item.Progress, &item.SizeBytes, &item.ErrorMessage,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM media_items WHERE job_id = $1`
	countArgs := []interface{}{jobID}
	if status != "" {
		countQuery += ` AND status = $2`
		countArgs = append(countArgs, status)
	}
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *Repository) UpdateMediaItemStatus(ctx context.Context, itemID, status string, progress int, errorMsg string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE media_items SET status = $1, progress = $2, error_message = $3, updated_at = NOW()
		WHERE id = $4
	`, status, progress, errorMsg, itemID)
	return err
}

func (r *Repository) UpdateMediaItemSize(ctx context.Context, itemID string, size int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE media_items SET size_bytes = $1, updated_at = NOW()
		WHERE id = $2
	`, size, itemID)
	return err
}

func (r *Repository) UpdateMediaItemsStatus(ctx context.Context, jobID, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE media_items SET status = $1, updated_at = NOW()
		WHERE job_id = $2 AND status NOT IN ('COMPLETED', 'FAILED')
	`, status, jobID)
	return err
}
