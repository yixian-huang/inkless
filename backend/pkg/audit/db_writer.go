package audit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/logger"
)

// Writer persists structured audit events.
type Writer interface {
	Write(ctx context.Context, event Event) error
}

// DbWriter persists audit events to the database
type DbWriter struct {
	repo repository.AuditEventRepository
	log  *logger.Logger
}

// NewDbWriter creates a new database-backed audit writer
func NewDbWriter(repo repository.AuditEventRepository, logs ...*logger.Logger) *DbWriter {
	var log *logger.Logger
	if len(logs) > 0 {
		log = logs[0]
	}
	return &DbWriter{repo: repo, log: log}
}

// Write persists a structured audit event to the database.
func (w *DbWriter) Write(ctx context.Context, event Event) error {
	if w == nil || w.repo == nil {
		return errors.New("audit repository is not configured")
	}

	detailsJSON := ""
	if event.Details != nil {
		b, err := json.Marshal(event.Details)
		if err != nil {
			w.reportError(event, err)
			return err
		}
		detailsJSON = string(b)
	}

	ae := &model.AuditEvent{
		Action:   event.Action,
		Actor:    event.Actor,
		Resource: event.Resource,
		Result:   event.Result,
		Details:  detailsJSON,
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := w.repo.Create(ctx, ae); err != nil {
		w.reportError(event, err)
		return err
	}
	return nil
}

// Log preserves the original best-effort API for existing callers.
func (w *DbWriter) Log(event Event) {
	_ = w.Write(context.Background(), event)
}

func (w *DbWriter) reportError(event Event, err error) {
	if w.log == nil {
		return
	}
	w.log.Error(
		"failed to persist audit event",
		"action", event.Action,
		"actor", event.Actor,
		"resource", event.Resource,
		"result", event.Result,
		"error", err,
	)
}

// LogPublishSuccess records a successful publish operation
func (w *DbWriter) LogPublishSuccess(pageKey string, publishedVersion int, actor string, draftVersion int) {
	w.Log(Event{
		Action:   "content.publish",
		Actor:    actor,
		Resource: pageKey,
		Result:   "success",
		Details: map[string]interface{}{
			"published_version": publishedVersion,
			"draft_version":     draftVersion,
		},
	})
}

// LogPublishFailure records a failed publish operation
func (w *DbWriter) LogPublishFailure(pageKey string, actor string, reason string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["reason"] = reason
	w.Log(Event{
		Action:   "content.publish",
		Actor:    actor,
		Resource: pageKey,
		Result:   "failure",
		Details:  details,
	})
}

// LogRollbackSuccess records a successful rollback operation
func (w *DbWriter) LogRollbackSuccess(pageKey string, publishedVersion int, sourceVersion int, actor string) {
	w.Log(Event{
		Action:   "content.rollback",
		Actor:    actor,
		Resource: pageKey,
		Result:   "success",
		Details: map[string]interface{}{
			"published_version": publishedVersion,
			"source_version":    sourceVersion,
		},
	})
}

// LogRollbackFailure records a failed rollback operation
func (w *DbWriter) LogRollbackFailure(pageKey string, actor string, sourceVersion int, reason string) {
	w.Log(Event{
		Action:   "content.rollback",
		Actor:    actor,
		Resource: pageKey,
		Result:   "failure",
		Details: map[string]interface{}{
			"source_version": sourceVersion,
			"reason":         reason,
		},
	})
}

// LogValidation records a validation operation
func (w *DbWriter) LogValidation(pageKey string, actor string, valid bool, errorCount int, translationIssueCount int) {
	result := "success"
	if !valid {
		result = "failure"
	}
	w.Log(Event{
		Action:   "content.validate",
		Actor:    actor,
		Resource: pageKey,
		Result:   result,
		Details: map[string]interface{}{
			"valid":                   valid,
			"error_count":             errorCount,
			"translation_issue_count": translationIssueCount,
		},
	})
}
