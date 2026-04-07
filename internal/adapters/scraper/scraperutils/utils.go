package scraperutils

import (
	"context"
	"sync"
)

func RepeatFunc[T any](ctx context.Context, fn func() T) <-chan T {
	stream := make(chan T)
	go func() {
		defer close(stream)
		for {
			select {
			case <-ctx.Done():
				return
			case stream <- fn():
			}
		}
	}()
	return stream
}

func Take[T any](ctx context.Context, stream <-chan T, n int) <-chan T {
	taken := make(chan T, n)
	go func() {
		defer close(taken)
		for range n {
			select {
			case <-ctx.Done():
				return
			case taken <- <-stream:
			}
		}
	}()
	return taken
}

func FanOut[T any, R any](ctx context.Context, input <-chan T, maxWorkers int, fn func(T) R) <-chan R {
	out := make(chan R, maxWorkers)
	sem := make(chan struct{}, maxWorkers)

	go func() {
		defer close(out)
		defer close(sem) // TODO: check if this is needed
		var wg sync.WaitGroup

		for item := range input {
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}

			wg.Add(1)
			go func(v T) {
				defer wg.Done()
				defer func() { <-sem }()
				result := fn(v)
				select {
				case <-ctx.Done():
				case out <- result:
				}
			}(item)
		}
		wg.Wait()
	}()

	return out
}
