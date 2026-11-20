package positions

import "time"

type realTickerFactory struct{}

type realTicker struct {
	ticker *time.Ticker
}

func (realTickerFactory) NewTicker(duration time.Duration) ticker {
	return realTicker{ticker: time.NewTicker(duration)}
}

func (t realTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t realTicker) Stop() {
	t.ticker.Stop()
}
