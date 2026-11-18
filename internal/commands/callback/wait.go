package callback

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// errTimeout is returned by waitUntilGone when the timeout expires.
var errTimeout = errors.New("timed out")

// waitUntilGone returns nil once nothing exists at path. It checks immediately
// and then every interval. A timeout of 0 waits indefinitely. It returns
// errTimeout when the timeout expires, ctx.Err() when ctx is done, and any
// other error from looking at the path. Only the absence of the path counts.
func waitUntilGone(ctx context.Context, path string, interval, timeout time.Duration) error {
	var expired <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		expired = timer.C
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return fmt.Errorf("checking notification file %q: %w", path, err)
		}
		select {
		case <-ticker.C:
		case <-expired:
			return errTimeout
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
