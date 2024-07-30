package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

var ErrDeleteJobFailed = errors.New("delete job failed")

func NewDeleteJobCommandHandler(repo jobs.Repository) app.Command[DeleteJobCommand] {
	return &deleteJobCommandHandler{repo: repo}
}

type deleteJobCommandHandler struct {
	repo jobs.Repository
}

type DeleteJobCommand struct {
	JobID string
}

func (h *deleteJobCommandHandler) H(ctx context.Context, cmd DeleteJobCommand) error {
	err := h.repo.DeleteByID(ctx, cmd.JobID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteJobFailed, err)
	}

	return nil
}
