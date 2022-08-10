// Package scheduler содержит реализацию планировщика, позволяющего запускать функции Go с заданными интервалами.
package scheduler

import (
	"context"
	"sync"
	"time"
)

// Scheduler позволяет запускать задачи с определенной периодичностью.
type Scheduler struct {
	wg            *sync.WaitGroup
	cancellations []context.CancelFunc
}

// New создает новый объект типа Scheduler.
func New() *Scheduler {
	return &Scheduler{
		wg:            new(sync.WaitGroup),
		cancellations: make([]context.CancelFunc, 0),
	}
}

// Add осуществляет вызов переданной функции f с периодом interval.
func (s *Scheduler) Add(ctx context.Context, f func(), interval time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancellations = append(s.cancellations, cancel)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f()
			}
		}
	}()
}

// Stop останавливает все задачи.
func (s *Scheduler) Stop() {
	for _, cancel := range s.cancellations {
		cancel()
	}
	s.wg.Wait()
}
