package app

import (
	"context"
	"fmt"

	"github.com/phin-tech/whisk/internal/domain/httpforward"
)

type HTTPForward = httpforward.Record

type CreateHTTPForwardRequest struct {
	Name      string
	TargetURL string
	SessionID string
}

func (r *Runtime) CreateHTTPForward(_ context.Context, req CreateHTTPForwardRequest) (HTTPForward, error) {
	id := r.ids()
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.forwards.Create(httpforward.CreateRequest{
		ID:        id,
		Name:      req.Name,
		TargetURL: req.TargetURL,
		SessionID: req.SessionID,
	})
}

func (r *Runtime) ListHTTPForwards(_ context.Context) ([]HTTPForward, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.forwards.List(), nil
}

func (r *Runtime) GetHTTPForward(_ context.Context, id string) (HTTPForward, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.forwards.Get(id)
	if !ok {
		return HTTPForward{}, fmt.Errorf("http forward %s not found", id)
	}
	return record, nil
}

func (r *Runtime) DeleteHTTPForward(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.forwards.Delete(id) {
		return fmt.Errorf("http forward %s not found", id)
	}
	return nil
}
