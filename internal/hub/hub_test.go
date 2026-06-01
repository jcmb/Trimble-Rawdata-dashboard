package hub

import (
	"sync"
	"testing"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/model"
)

func TestPublishAfterUnsubscribeDoesNotPanic(t *testing.T) {
	h := New()
	ch := h.Subscribe()
	h.Unsubscribe(ch)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.PublishSnapshot(model.Snapshot{Connected: false})
		}()
	}
	wg.Wait()
}
