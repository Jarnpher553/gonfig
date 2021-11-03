package retry

import "time"

func Retry(count int, f func() error, d ...time.Duration) error {
	duration := time.Second * 5
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
