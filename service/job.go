package service

import (
	"context"
	"errors"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	"math/rand"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

const (
	maxAttempts       = 3
	retryDelay        = 1 * time.Second
	workDelay         = 2 * time.Second
	unstableFailTimes = 2
)

type jobService struct {
	jobRepo    _interface.JobRepository
	taskLocks  map[string]string
	locksMutex sync.RWMutex
}

// Initiator ...
type Initiator func(s *jobService) *jobService

func (q *jobService) GetAllJobs(ctx context.Context) ([]*entity.Job, error) {
	jobs, err := q.jobRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (q *jobService) FindByID(ctx context.Context, id string) (*entity.Job, error) {
	job, err := q.jobRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (q *jobService) GetJobStatus(ctx context.Context) (entity.JobStatus, error) {
	jobs, err := q.jobRepo.FindAll(ctx)
	if err != nil {
		return entity.JobStatus{}, err
	}
	var status entity.JobStatus
	for _, j := range jobs {
		switch j.Status {
		case constant.StatusPending:
			status.Pending++
		case constant.StatusRunning:
			status.Running++
		case constant.StatusFailed:
			status.Failed++
		case constant.StatusCompleted:
			status.Completed++
		}
	}
	return status, nil
}

func (q *jobService) Enqueue(ctx context.Context, taskName string) (string, error) {
	q.locksMutex.Lock()
	if existingID, ok := q.taskLocks[taskName]; ok {
		q.locksMutex.Unlock()
		existing, err := q.jobRepo.FindByID(ctx, existingID)
		if err == nil && (existing.Status == constant.StatusPending || existing.Status == constant.StatusRunning) {
			zap.S().Infow("job idempotent: returning existing id", "id", existingID, "task", taskName)
			return existingID, nil
		}
	} else {
		q.locksMutex.Unlock()
	}

	id := uuid.NewV4().String()

	q.locksMutex.Lock()
	q.taskLocks[taskName] = id
	q.locksMutex.Unlock()

	job := &entity.Job{
		ID:       id,
		Task:     taskName,
		Status:   constant.StatusPending,
		Attempts: 0,
	}

	if err := q.jobRepo.Save(ctx, job); err != nil {
		return "", err
	}

	zap.S().Infow("job enqueued", "id", id, "task", taskName)

	go q.processJob(id)

	return id, nil
}

func (q *jobService) processJob(jobID string) {
	ctx := context.Background()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		job, err := q.jobRepo.FindByID(ctx, jobID)
		if err != nil {
			zap.S().Errorw("processJob: job not found", "id", jobID, "err", err)
			return
		}

		job.Status = constant.StatusRunning
		if err := q.jobRepo.Save(ctx, job); err != nil {
			zap.S().Errorw("processJob: failed to set running", "id", jobID, "err", err)
			return
		}
		zap.S().Infow("job running", "id", jobID, "task", job.Task, "attempt", attempt)

		time.Sleep(workDelay)

		job.Attempts = int32(attempt)

		if err := q.executeJob(job); err != nil {
			if attempt < maxAttempts {
				job.Status = constant.StatusPending
				_ = q.jobRepo.Save(ctx, job)
				zap.S().Infow("job failed, will retry", "id", jobID, "task", job.Task, "attempt", attempt, "err", err)
				time.Sleep(retryDelay)
				continue
			}
			job.Status = constant.StatusFailed
			_ = q.jobRepo.Save(ctx, job)
			zap.S().Errorw("job failed permanently", "id", jobID, "task", job.Task, "attempts", attempt, "err", err)
			q.releaseLock(job.Task)
			return
		}

		job.Status = constant.StatusCompleted
		if err := q.jobRepo.Save(ctx, job); err != nil {
			zap.S().Errorw("processJob: failed to set completed", "id", jobID, "err", err)
		}
		zap.S().Infow("job completed", "id", jobID, "task", job.Task, "attempt", attempt)
		q.releaseLock(job.Task)
		return
	}
}

func (q *jobService) executeJob(job *entity.Job) error {
	if job.Task == "unstable-job" {
		if job.Attempts <= unstableFailTimes {
			zap.S().Infow("unstable-job simulating failure", "id", job.ID, "attempt", job.Attempts)
			return errors.New("simulated unstable failure")
		}
	} else {
		if rand.Intn(100) < 10 {
			return errors.New("simulated random failure")
		}
	}
	return nil
}

func (q *jobService) releaseLock(taskName string) {
	q.locksMutex.Lock()
	defer q.locksMutex.Unlock()
	delete(q.taskLocks, taskName)
}

// NewJobService ...
func NewJobService() Initiator {
	return func(s *jobService) *jobService {
		s.taskLocks = make(map[string]string)
		return s
	}
}

// SetJobRepository ...
func (i Initiator) SetJobRepository(jobRepository _interface.JobRepository) Initiator {
	return func(s *jobService) *jobService {
		i(s).jobRepo = jobRepository
		return s
	}
}

// Build ...
func (i Initiator) Build() _interface.JobService {
	return i(&jobService{})
}
