package runner

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/Chocapikk/pik/pkg/core"
	pikhttp "github.com/Chocapikk/pik/pkg/http"
	"github.com/Chocapikk/pik/pkg/output"
)

// --- Types ---

// Result holds the outcome of scanning a single target.
type Result struct {
	Target     string
	Vulnerable bool
	Error      error
}

// Scanner runs vulnerability checks against multiple targets concurrently.
type Scanner struct {
	Module     core.Exploit
	Targets    []string
	Threads    int
	BaseParams core.Params
	OutputFile string
}

// --- Scan execution ---

// Run checks all targets and returns results.
func (s *Scanner) Run(ctx context.Context) []Result {
	checker, ok := s.Module.(core.Checker)
	if !ok {
		output.Warning("Module %s does not support check", core.NameOf(s.Module))
		return nil
	}

	ctx = pikhttp.WithPool(ctx, s.Threads)
	total := int64(len(s.Targets))

	results := make([]Result, total)
	var done int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.Threads)

	for i, target := range s.Targets {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, addr string) {
			defer wg.Done()
			defer func() { <-sem }()

			result := s.checkTarget(ctx, checker, addr)
			results[idx] = result

			cur := atomic.AddInt64(&done, 1)
			s.logResult(result, cur, total)
		}(i, target)
	}

	wg.Wait()
	s.logSummary(results, total)

	if s.OutputFile != "" {
		s.writeVulnerable(results)
	}

	return results
}

// --- Internal ---

func (s *Scanner) checkTarget(ctx context.Context, checker core.Checker, addr string) Result {
	params := s.BaseParams.Clone()
	params.Ctx = ctx
	params.Set("TARGET", addr)

	run := buildContext(params, "")
	check, err := checker.Check(run)

	return Result{
		Target:     addr,
		Vulnerable: err == nil && check.Code.IsVulnerable(),
		Error:      err,
	}
}

func (s *Scanner) logResult(r Result, cur, total int64) {
	switch {
	case r.Error != nil:
		output.Error("%s - %v", r.Target, r.Error)
	case r.Vulnerable:
		output.Success("%s - vulnerable", r.Target)
	}

	if cur == total || cur%25 == 0 {
		output.Status("Progress: %d/%d (%d%%)", cur, total, cur*100/total)
	}
}

func (s *Scanner) logSummary(results []Result, total int64) {
	var vuln, errs int
	for _, r := range results {
		if r.Vulnerable {
			vuln++
		}
		if r.Error != nil {
			errs++
		}
	}
	output.Success("Scan complete: %d targets, %d vulnerable, %d errors", total, vuln, errs)
}

func (s *Scanner) writeVulnerable(results []Result) {
	f, err := os.Create(s.OutputFile)
	if err != nil {
		output.Error("Failed to write %s: %v", s.OutputFile, err)
		return
	}
	defer f.Close()

	count := 0
	for _, r := range results {
		if r.Vulnerable {
			fmt.Fprintln(f, r.Target)
			count++
		}
	}
	output.Status("Wrote %d vulnerable targets to %s", count, s.OutputFile)
}
