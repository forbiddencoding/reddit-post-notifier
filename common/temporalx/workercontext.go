package temporalx

import "context"

func WorkerInterruptFromCtxChan(ctx context.Context) <-chan any {
	ch := make(chan any, 1)

	go func() {
		defer close(ch)
		<-ctx.Done()
	}()

	return ch
}
