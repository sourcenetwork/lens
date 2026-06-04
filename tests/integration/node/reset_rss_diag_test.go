// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Build-tagged `leakdiag` so it is invisible to the normal suite (`go test ./...`,
// `make test`, `make test:ci`). It is an on-demand diagnostic, not a CI gate — run it
// with `make diag:reset-rss` (issue #160).
//go:build leakdiag

package node

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/node"
	"github.com/sourcenetwork/lens/host-go/store"
	"github.com/sourcenetwork/lens/tests/modules"

	"github.com/stretchr/testify/require"
)

const (
	rssChildEnv  = "LENS_RSS_CHILD"
	warmupCycles = 20_000
	// High volume + long enough that a healthy run reaches a bounded plateau well before
	// the end, so the tail reflects steady state rather than the warm-up ramp.
	measuredCycles = 500_000
	progressEvery  = 20_000
	tailIntervals  = 3
	// A converged (non-leaking) run barely moves over its final intervals; an unbounded
	// leak keeps climbing. Allow this much RSS increase across the last `tailIntervals`
	// progress intervals before calling it "still climbing". Generous — this is a smoke
	// alarm for gross regressions, not a tight threshold.
	maxTailGrowthKiB = 32 * 1024 // 32 MiB over the last ~60k cycles
)

type rssSample struct {
	cycle int
	rss   int
}

// TestResetRSSDiagnostic drives a high volume of Store.Transform/Reset cycles through the
// full node->store->repository->engine->runtime stack and watches the process's resident
// memory (RSS) for *unbounded* growth (issue #160 / #155).
//
// At the integration layer the wasm instance is hidden behind the repository pool, so
// instance.Memory().Size() is unavailable and RSS is the only signal. The workload runs
// in a child process; the parent samples the child's RSS at the post-GC checkpoints the
// child emits and reports the trend. The verdict is *convergence*: a healthy run plateaus
// (tail growth ~0), a leak keeps climbing. It always logs the trend and only fails on a
// still-climbing tail — it is a diagnostic, not a guarantee of leak-freedom.
func TestResetRSSDiagnostic(t *testing.T) {
	if testing.Short() {
		t.Skip("rss leak diagnostic skipped under -short")
	}

	// Child: run the workload and exit. Markers go to stdout (fmt, not t.Log, which is
	// buffered until the test ends) so the parent can read them live.
	if os.Getenv(rssChildEnv) == "1" {
		runResetWorkload(t)
		return
	}

	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skipf("rss sampling unsupported on %s", runtime.GOOS)
	}

	// Parent: spawn the child as a fresh process and sample its RSS on each checkpoint.
	cmd := exec.Command(os.Args[0], "-test.run=^TestResetRSSDiagnostic$", "-test.v")
	cmd.Env = append(os.Environ(), rssChildEnv+"=1")
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	pid := cmd.Process.Pid

	sampleNow := func() int {
		rss, err := rssKiB(pid)
		if err != nil {
			return -1
		}
		return rss
	}

	// The child GC+frees right before each marker, so every sample is a retained-footprint
	// (post-collection) reading at a known cycle count.
	var samples []rssSample
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		t.Logf("child | %s", line)
		switch {
		case strings.Contains(line, "WARMUP_COMPLETE"):
			if rss := sampleNow(); rss >= 0 {
				samples = append(samples, rssSample{0, rss})
			}
		case strings.HasPrefix(line, "PROGRESS "):
			if n, perr := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "PROGRESS"))); perr == nil {
				if rss := sampleNow(); rss >= 0 {
					samples = append(samples, rssSample{n, rss})
				}
			}
		case strings.Contains(line, "DONE"):
			if rss := sampleNow(); rss >= 0 {
				samples = append(samples, rssSample{measuredCycles, rss})
			}
		}
	}
	require.NoError(t, cmd.Wait(), "child workload failed (see child output above)")
	require.GreaterOrEqual(t, len(samples), tailIntervals+1, "too few RSS samples collected")

	for _, s := range samples {
		t.Logf("cycle=%-7d rss=%d KiB", s.cycle, s.rss)
	}
	baseline := samples[0].rss
	final := samples[len(samples)-1].rss
	tail := final - samples[len(samples)-1-tailIntervals].rss
	t.Logf("RSS: baseline=%d KiB  final=%d KiB  total-growth=%d KiB; tail-growth over last %d cycles=%d KiB (converged if <= %d)",
		baseline, final, final-baseline, tailIntervals*progressEvery, tail, maxTailGrowthKiB)

	if tail > maxTailGrowthKiB {
		t.Fatalf("RSS did not converge: it grew %d KiB over the final ~%d cycles and is still climbing — likely a leak; inspect the trend above",
			tail, tailIntervals*progressEvery)
	}
}

// runResetWorkload (child) drives the high-volume Transform/Reset loop through a real node.
func runResetWorkload(t *testing.T) {
	ctx := context.Background()

	// Build the node directly (the testo action harness wraps execution in a ~1s context
	// that cannot sustain high volume). PoolSize(1) forces the same instance to be reset
	// every cycle — the #155 scenario.
	n, err := node.New(ctx, node.WithPath(t.TempDir()), node.WithPoolSize(1))
	require.NoError(t, err)
	defer func() { _ = n.Close() }()

	id, err := n.Store.Add(ctx, model.Lens{
		Lenses: []model.LensModule{{Path: modules.WasmPath1}},
	})
	require.NoError(t, err)

	cycle := func(age int) {
		src := enumerable.New([]store.Document{{"Name": "John", "Age": age}})
		out, err := n.Store.Transform(ctx, src, id)
		require.NoError(t, err)
		for {
			hasNext, err := out.Next()
			require.NoError(t, err)
			if !hasNext {
				break
			}
			_, err = out.Value()
			require.NoError(t, err)
		}
		out.Reset()
	}

	// settle prints a post-GC marker so the parent samples retained (not transient) memory.
	settle := func(marker string) {
		runtime.GC()
		debug.FreeOSMemory()
		fmt.Println(marker)
	}

	for i := 0; i < warmupCycles; i++ {
		cycle(i)
	}
	settle("WARMUP_COMPLETE")

	for i := 1; i <= measuredCycles; i++ {
		cycle(warmupCycles + i)
		if i%progressEvery == 0 {
			settle(fmt.Sprintf("PROGRESS %d", i))
		}
	}
	settle("DONE")
	// Stay alive briefly so the parent can sample the final post-GC footprint before exit.
	time.Sleep(time.Second)
}

// rssKiB returns the resident set size of the given process in KiB.
func rssKiB(pid int) (int, error) {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
		if err != nil {
			return 0, err
		}
		fields := strings.Fields(string(data))
		if len(fields) < 2 {
			return 0, fmt.Errorf("unexpected /proc/%d/statm format", pid)
		}
		pages, err := strconv.Atoi(fields[1]) // resident pages
		if err != nil {
			return 0, err
		}
		return pages * os.Getpagesize() / 1024, nil
	case "darwin":
		out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return 0, err
		}
		v, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			return 0, err
		}
		return v, nil // ps reports KiB
	default:
		return 0, fmt.Errorf("rss sampling unsupported on %s", runtime.GOOS)
	}
}
