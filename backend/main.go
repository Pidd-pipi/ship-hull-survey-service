package main

import (
	"context"
	"net/http"
	"sync"
	"time"
)

func main() {
	c := loadConfig()
	deps := newAppDeps()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	scanner := newScanner(deps.ops.store, deps.clock)
	scanner.Start(ctx, 5*time.Minute, &wg)
	m := newAppHandler(deps, scanner)
	if e := serveAddress(":"+c.Port, m); e != nil {
		cancel()
		wg.Wait()
		panic(e)
	}
	cancel()
	wg.Wait()
}

// appDeps holds the shared in-memory state of the service.
type appDeps struct {
	findings *FindingStore
	ops      *OpsService
	evidence *EvidenceStore
	dock     *DryDock
	clock    OpsClock
}

func newAppDeps() *appDeps {
	clock := newOpsClock()
	return &appDeps{
		findings: newFindingStore(),
		ops:      newOpsService(opsSeed()),
		evidence: newEvidenceStore(),
		dock:     newDryDock(clock),
		clock:    clock,
	}
}

// newAppHandler wires the survey findings router, the operations workflow,
// the evidence attachments, the report endpoint and the dry-dock planner
// behind one http.Handler used by the production server.
func newAppHandler(deps *appDeps, scanner *Scanner) http.Handler {
	m := http.NewServeMux()
	m.Handle("/", newRouter(deps.findings))
	m.Handle("/api/ops/", newOpsHandler(deps.ops))
	m.Handle("/api/ops/scanner/", newScannerHandler(scanner))
	m.Handle("/api/evidence/", newEvidenceHandler(deps.findings, deps.evidence))
	m.Handle("/api/report/", newReportHandler(deps.findings, deps.clock))
	m.Handle("/api/drydock/", newDryDockHandler(deps.dock))
	return m
}
