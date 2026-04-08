package common

import (
	"context"
	"errors"
	"net"
	"time"
)

const (
	ErrTypeAddContextToConnection = "add context to connection"
)

var ErrWrapperAddContextToConnection = NewErrorWrapper(ErrTypeAddContextToConnection)

func AddContextToConnection(connection net.Conn, ctx context.Context) WrappedError {
	deadline, ok := ctx.Deadline()
	if ok {
		stdErr := connection.SetDeadline(deadline)
		if stdErr != nil {
			return ErrWrapperAddContextToConnection.Wrap(stdErr)
		}
	}

	doneChan := ctx.Done()
	if doneChan != nil {
		go func() {
			<-doneChan
			stdErr := ctx.Err()
			if !errors.Is(stdErr, context.DeadlineExceeded) {
				_ = connection.SetDeadline(time.Now())
				_ = connection.Close()
			}
		}()
	}

	return nil
}
