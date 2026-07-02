// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ltmbeverifyshare implements a live verification step for the WS4b MBE traffic-steering
// target share (see mbe-load-test-plan.md WS4b). It queries the deployment's Prometheus
// instance directly for the actual, currently-observed fraction of write-path (post/update)
// and read-path (consume) hook invocations that targeted an MBE channel, using the plugin's
// is_mbe label -- the same quantity switchChannel's MBEChannelWeight is steering toward -- and
// compares it against the configured target.
//
// This is a fast, informational check meant to be run repeatedly against a small deployment
// while tuning MBEChannelWeight (M2), and again during full ramps (M4/M5) as an early warning
// alongside the mandatory post-run DB validation. It does not replace that DB check: live
// metrics can drift from ground truth (e.g. under Prometheus scrape gaps), and only the DB
// tells you exactly how many posts landed in MBE channels over the whole run window.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
)

func main() {
	var (
		prometheusURL = flag.String("prometheus-url", "", "Prometheus server URL, e.g. http://1.2.3.4:9090 (required)")
		expected      = flag.Float64("expected", 0, "expected MBE target share, matching the run's MBEChannelWeight (0, 0.1, or 1)")
		tolerance     = flag.Float64("tolerance", 0.2, "allowed relative deviation from -expected (ignored when -expected is 0)")
		toleranceAbs  = flag.Float64("tolerance-abs", 0.01, "allowed absolute deviation from -expected when -expected is 0")
		window        = flag.String("window", "5m", "Prometheus rate() window")
	)
	flag.Parse()

	if *prometheusURL == "" {
		fmt.Fprintln(os.Stderr, "-prometheus-url is required")
		flag.Usage()
		os.Exit(2)
	}

	helper, err := prometheus.NewHelper(*prometheusURL)
	if err != nil {
		log.Fatalf("failed to create Prometheus client: %v", err)
	}

	writeShare, err := shareQuery(helper, "post|update", *window)
	if err != nil {
		log.Fatalf("write-path query failed: %v", err)
	}

	readShare, err := shareQuery(helper, "consume", *window)
	if err != nil {
		log.Fatalf("read-path query failed: %v", err)
	}

	log.Printf("write-path MBE share: %.4f (expected %.4f)", writeShare, *expected)
	log.Printf("read-path MBE share:  %.4f (expected %.4f)", readShare, *expected)

	writeOK := withinTolerance(writeShare, *expected, *tolerance, *toleranceAbs)
	readOK := withinTolerance(readShare, *expected, *tolerance, *toleranceAbs)
	if !writeOK || !readOK {
		log.Fatalf("MBE share deviates from expected %.4f: write=%.4f (ok=%v) read=%.4f (ok=%v)",
			*expected, writeShare, writeOK, readShare, readOK)
	}

	log.Print("MBE share within tolerance for both write and read paths")
}

// shareQuery returns the fraction of mattermost_plugin_mbe_hook_invocations_total for the given
// hook pattern that carry is_mbe="true", i.e. the live-observed traffic-steering target share.
func shareQuery(helper *prometheus.Helper, hookPattern, window string) (float64, error) {
	query := fmt.Sprintf(
		`sum(rate(mattermost_plugin_mbe_hook_invocations_total{hook=~"%s",is_mbe="true"}[%s])) `+
			`/ sum(rate(mattermost_plugin_mbe_hook_invocations_total{hook=~"%s"}[%s]))`,
		hookPattern, window, hookPattern, window,
	)
	value, err := helper.VectorFirst(query)
	if err != nil {
		return 0, fmt.Errorf("query %q: %w", query, err)
	}
	return value, nil
}

func withinTolerance(actual, expected, relTolerance, absTolerance float64) bool {
	if expected == 0 {
		return actual <= absTolerance
	}
	deviation := (actual - expected) / expected
	if deviation < 0 {
		deviation = -deviation
	}
	return deviation <= relTolerance
}
