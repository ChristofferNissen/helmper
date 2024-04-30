package counter

import (
	"sync"
	"testing"
)

func TestSafeCounter(t *testing.T) {
	s := NewSafeCounter()
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			s.Inc("test")
			wg.Done()
		}()
	}
	wg.Wait()

	expected := 1000

	if s.Value("test") != expected {
		t.Errorf("want '%d' got '%d'", expected, s.Value("test"))
	}
}
