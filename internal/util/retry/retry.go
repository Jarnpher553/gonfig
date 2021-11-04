package retry

import "time"

// Retry retry several times to exec func with a interval
func Retry(count int, f func() error, d ...time.Duration) error {
	duration := time.Second * 1
	if len(d) != 0 {
		duration = d[0]
	}
	t := time.NewTimer(duration)
	var err error
	for i := 0; i < count; i++ {
		t.Reset(duration)
		if err = f(); err != nil {
			<-t.C
			continue
		}

		return nil
	}

	return err
}
