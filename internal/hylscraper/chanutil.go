package hylscraper

import "context"

func repeatFunc[T any](ctx context.Context, fn func() T) <-chan T {
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

func take[T any](ctx context.Context, stream <-chan T, n int) <-chan T {
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
