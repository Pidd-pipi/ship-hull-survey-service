package main

import (
	"sync"
	"testing"
)

func TestAuditAddForConcurrentRace(t *testing.T) {
	a := newOpsAudit()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				a.Add("hull-1001", "status_changed", "tester")
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				_ = a.For("hull-1001")
				_ = a.Count()
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := a.Count(); got != 6*50 {
		t.Fatalf("count = %d, want %d", got, 6*50)
	}
}
