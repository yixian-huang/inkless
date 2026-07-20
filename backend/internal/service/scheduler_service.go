package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/pkg/audit"
)

const (
	defaultClaimLimit    = 10
	defaultLeaseDuration = 2 * time.Minute
)

type SchedulerService struct {
	jobRepo   repository.ScheduledPublishJobRepository
	kernel    *PublicationKernel
	logger    *slog.Logger
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

// NewSchedulerService wires the durable schedule queue onto the publication
// kernel. Pass any ContentPublisher implementations (article, page, …).
func NewSchedulerService(
	jobRepo repository.ScheduledPublishJobRepository,
	publishers ...ContentPublisher,
) *SchedulerService {
	return &SchedulerService{
		jobRepo: jobRepo,
		kernel:  NewPublicationKernel(publishers...),
		logger:  slog.Default(),
		done:    make(chan struct{}),
	}
}

// NewSchedulerServiceWithKernel is useful in tests that inject a custom kernel.
func NewSchedulerServiceWithKernel(
	jobRepo repository.ScheduledPublishJobRepository,
	kernel *PublicationKernel,
) *SchedulerService {
	return &SchedulerService{
		jobRepo: jobRepo,
		kernel:  kernel,
		logger:  slog.Default(),
		done:    make(chan struct{}),
	}
}

func (s *SchedulerService) Schedule(
	ctx context.Context,
	contentType model.ScheduledContentType,
	contentID uint,
	scheduledAt time.Time,
	expectedVersion *int,
	publishPayload model.JSONMap,
	actorID uint,
) (*model.ScheduledPublishJob, error) {
	resolvedVersion, expectedUpdatedAt, err := s.prepareSchedule(ctx, contentType, contentID, expectedVersion, nil, publishPayload)
	if err != nil {
		return nil, err
	}
	job := &model.ScheduledPublishJob{
		ContentType:       contentType,
		ContentID:         contentID,
		Status:            model.ScheduledJobPending,
		ScheduledAt:       scheduledAt,
		ExpectedVersion:   resolvedVersion,
		ExpectedUpdatedAt: expectedUpdatedAt,
		PublishPayload:    publishPayload,
		MaxAttempts:       3,
		CreatedBy:         actorID,
		UpdatedBy:         actorID,
	}
	if err := s.jobRepo.Schedule(ctx, job); err != nil {
		return nil, err
	}
	// Note: do not call MarkScheduled here. Resource-row writes (status /
	// scheduled_at) bump UpdatedAt and would invalidate article concurrency
	// tokens captured in PrepareSchedule. The durable job is the source of
	// truth; handlers may call ContentPublisher.MarkScheduled explicitly when
	// they need resource-level UX status without an intervening lock.
	return job, nil
}

func (s *SchedulerService) Reschedule(
	ctx context.Context,
	jobID uint,
	scheduledAt time.Time,
	expectedVersion *int,
	publishPayload model.JSONMap,
	actorID uint,
) (*model.ScheduledPublishJob, error) {
	current, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	pub, err := s.kernel.Publisher(current.ContentType)
	if err != nil {
		return nil, err
	}
	resolvedInputVersion, expectedUpdatedAt, resolvedPayload := pub.MergeRescheduleHints(
		current.ExpectedVersion,
		current.ExpectedUpdatedAt,
		current.PublishPayload,
		expectedVersion,
		publishPayload,
	)
	resolvedVersion, expectedUpdatedAt, err := s.prepareSchedule(
		ctx,
		current.ContentType,
		current.ContentID,
		resolvedInputVersion,
		expectedUpdatedAt,
		resolvedPayload,
	)
	if err != nil {
		return nil, err
	}
	job, err := s.jobRepo.UpdateSchedule(
		ctx,
		jobID,
		scheduledAt,
		resolvedVersion,
		expectedUpdatedAt,
		resolvedPayload,
		actorID,
	)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *SchedulerService) Cancel(ctx context.Context, jobID uint, actorID uint, cancelledAt time.Time) (*model.ScheduledPublishJob, error) {
	return s.jobRepo.Cancel(ctx, jobID, actorID, cancelledAt)
}

func (s *SchedulerService) Retry(
	ctx context.Context,
	jobID uint,
	actorID uint,
	retryAt time.Time,
) (*model.ScheduledPublishJob, error) {
	current, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	resolvedVersion, _, err := s.prepareSchedule(
		ctx,
		current.ContentType,
		current.ContentID,
		current.ExpectedVersion,
		current.ExpectedUpdatedAt,
		current.PublishPayload,
	)
	if err != nil {
		return nil, err
	}
	job, err := s.jobRepo.Retry(ctx, jobID, retryAt, resolvedVersion, actorID)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *SchedulerService) Get(ctx context.Context, jobID uint) (*model.ScheduledPublishJob, error) {
	return s.jobRepo.FindByID(ctx, jobID)
}

func (s *SchedulerService) List(
	ctx context.Context,
	contentTypes []model.ScheduledContentType,
	contentID uint,
	status model.ScheduledJobStatus,
	offset,
	limit int,
) ([]*model.ScheduledPublishJob, int64, error) {
	return s.jobRepo.List(ctx, contentTypes, contentID, status, offset, limit)
}

func (s *SchedulerService) DescribeResource(
	ctx context.Context,
	job *model.ScheduledPublishJob,
) (title string, slug string) {
	if job == nil {
		return "", ""
	}
	return s.kernel.Describe(ctx, job.ContentType, job.ContentID, job.PublishPayload)
}

// PublishOverdue preserves the old public method name while using the durable queue.
func (s *SchedulerService) PublishOverdue(ctx context.Context) (int, error) {
	return s.RunDue(ctx, time.Now())
}

func (s *SchedulerService) RunDue(ctx context.Context, now time.Time) (int, error) {
	jobs, err := s.jobRepo.ClaimDue(ctx, now, defaultClaimLimit, defaultLeaseDuration)
	if err != nil {
		return 0, err
	}
	succeeded := 0
	var errs []error
	for _, job := range jobs {
		if err := s.runJob(ctx, job, now); err != nil {
			errs = append(errs, fmt.Errorf("job %d: %w", job.ID, err))
			continue
		}
		succeeded++
	}
	return succeeded, errors.Join(errs...)
}

func (s *SchedulerService) runJob(ctx context.Context, job *model.ScheduledPublishJob, now time.Time) error {
	ctx = audit.WithMetadata(ctx, audit.Metadata{
		Actor:   "scheduler",
		ActorID: job.CreatedBy,
	})
	err := s.kernel.ExecuteScheduled(
		ctx,
		job.ContentType,
		job.ContentID,
		now,
		job.CreatedBy,
		job.ExpectedVersion,
		job.ExpectedUpdatedAt,
		job.PublishPayload,
	)
	if err != nil {
		retryAt := now.Add(time.Duration(job.Attempts) * time.Minute)
		if markErr := s.jobRepo.MarkFailed(ctx, job, err.Error(), retryAt, now); markErr != nil {
			return errors.Join(err, fmt.Errorf("mark failed: %w", markErr))
		}
		return err
	}
	if err := s.jobRepo.MarkSucceeded(ctx, job, now); err != nil {
		return fmt.Errorf("mark succeeded: %w", err)
	}
	return nil
}

func (s *SchedulerService) prepareSchedule(
	ctx context.Context,
	contentType model.ScheduledContentType,
	contentID uint,
	expectedVersion *int,
	expectedUpdatedAt *time.Time,
	publishPayload model.JSONMap,
) (*int, *time.Time, error) {
	return s.kernel.PrepareSchedule(
		ctx,
		contentType,
		contentID,
		expectedVersion,
		expectedUpdatedAt,
		publishPayload,
	)
}

func (s *SchedulerService) Start() {
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runTick(time.Now())
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-s.done:
					return
				case tickAt := <-ticker.C:
					s.runTick(tickAt)
				}
			}
		}()
		s.logger.Info("Scheduler started", "interval", "1m")
	})
}

func (s *SchedulerService) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)
		s.wg.Wait()
		s.logger.Info("Scheduler stopped")
	})
}

func (s *SchedulerService) runTick(tickAt time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.RunDue(ctx, tickAt); err != nil {
		s.logger.Error("Scheduler error", "error", err)
	}
}
