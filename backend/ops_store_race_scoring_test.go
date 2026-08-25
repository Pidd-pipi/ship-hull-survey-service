package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestOpsStoreConcurrentAccess(t *testing.T) {
	store := newOpsStore(opsSeed())
	const workers = 8
	const perWorker = 20
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			for j := 0; j < perWorker; j++ {
				rec := OpsRecord{ID: fmt.Sprintf("hull-x-%d-%d", n, j), Subject: "concurrent sweep", Owner: "tester", Status: OpsStatusQueued, Priority: OpsPriorityLow, Labels: map[string]string{"site": "MV Test", "operator": "tester", "evidence": "e1"}}
				if err := store.Put(context.Background(), rec); err != nil {
					t.Errorf("put: %v", err)
				}
				if _, err := store.Get(context.Background(), "hull-1001"); err != nil {
					t.Errorf("get: %v", err)
				}
				if _, err := store.List(context.Background()); err != nil {
					t.Errorf("list: %v", err)
				}
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if got := store.Count(); got != 3+workers*perWorker {
		t.Fatalf("count = %d, want %d", got, 3+workers*perWorker)
	}
}

func TestOpsListGetConcurrentPutRace(t *testing.T) {
	store := newOpsStore(opsSeed())
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				rec := OpsRecord{ID: fmt.Sprintf("hull-r-%d-%d", n, j), Subject: "race", Owner: "tester", Status: OpsStatusActive, Priority: OpsPriorityNormal, Labels: map[string]string{"site": "MV R", "operator": "tester", "evidence": "e2"}}
				_ = store.Put(context.Background(), rec)
			}
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				_, _ = store.List(context.Background())
				_ = store.Count()
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestOpsReadsReturnCopies(t *testing.T) {
	store := newOpsStore(opsSeed())
	rec, err := store.Get(context.Background(), "hull-1001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	rec.Labels["site"] = "mutated"
	got, _ := store.Get(context.Background(), "hull-1001")
	if got.Labels["site"] != "MV Aurora" {
		t.Fatalf("store leaked internal reference, site=%q", got.Labels["site"])
	}
	items, _ := store.List(context.Background())
	items[0].Labels["operator"] = "hacked"
	again, _ := store.Get(context.Background(), items[0].ID)
	if again.Labels["operator"] == "hacked" {
		t.Fatalf("list leaked internal reference for %s", items[0].ID)
	}
}
