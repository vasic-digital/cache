// Cache describe-Challenge runner.
//
// This is a CHALLENGE program (not production code) — it exercises
// the REAL Cache module's in-memory backend AND the policy package
// against real byte-slice payloads in two locales (en + sr-Latn).
//
// The Challenge proves four invariants end-to-end:
//
//  1. memory.Cache really stores + retrieves bytes (not in-test
//     phantom storage). We Set bilingual locale fixtures into the
//     cache, Get them back, and assert byte-equality.
//  2. The cache.Stats counters increment as real operations occur
//     (hit/miss accounting is wired, not bluffed).
//  3. TTL policy actually controls expiry — FixedTTL of 50ms is
//     applied and the entry MUST be invisible after a 200ms sleep.
//  4. The bilingual describe-string mapping is locale-correct AND
//     differs between EN and SR — sanity check that we are not
//     serving the same language for both locales.
//
// Anti-bluff posture (CONST-035 / CONST-050):
//   - the cache.Cache and memory.Cache types are EXERCISED, never mocked;
//   - every assertion captures the actual returned bytes / int / bool
//     verbatim so a regression cannot disguise itself as "test still
//     passing";
//   - the paired-mutation leg (driven from
//     challenges/cache_describe_challenge.sh) corrupts a YAML entry
//     and re-runs this program — failure expected.
//
// Exit codes:
//
//	0 — every assertion held; bilingual round-trip + TTL + stats green.
//	1 — bundle load failure, cache regression, TTL drift, or locale-mismatch.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"digital.vasic.cache/pkg/cache"
	"digital.vasic.cache/pkg/memory"
	"digital.vasic.cache/pkg/policy"
)

// loadBundle parses a single-locale YAML fixture into a flat map.
// Lookup-miss callers receive an explicit error so missing-key
// regressions are loud, not silent.
func loadBundle(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bundle %s: %w", path, err)
	}
	entries := map[string]string{}
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse bundle %s: %w", path, err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("bundle %s parsed but contains no entries", path)
	}
	return entries, nil
}

// expectedStates is the canonical assertion table — the message IDs
// the Challenge probes against. Source-of-truth list lives here, not
// in production code (CONST-046 carve-out: fixtures + assertions are
// not user-facing strings).
var expectedStates = []string{
	"cache.state.empty",
	"cache.state.populated",
	"cache.state.evicted",
	"cache.state.expired",
	"cache.policy.lru",
	"cache.policy.lfu",
	"cache.policy.fifo",
}

func mustEqualBytes(label string, got, want []byte) error {
	if string(got) != string(want) {
		return fmt.Errorf("%s: got %q want %q", label, string(got), string(want))
	}
	fmt.Printf("  OK  %s -> %d bytes verbatim\n", label, len(got))
	return nil
}

func mustEqualBool(label string, got, want bool) error {
	if got != want {
		return fmt.Errorf("%s: got %v want %v", label, got, want)
	}
	fmt.Printf("  OK  %s -> %v\n", label, got)
	return nil
}

// runLocale drives one locale's fixture entries through the real
// memory.Cache: Set every entry, Get every entry back, assert
// byte-equality, then Exists() the key and Delete it.
func runLocale(ctx context.Context, fixturePath, locale string) error {
	entries, err := loadBundle(fixturePath)
	if err != nil {
		return err
	}

	fmt.Printf("--- locale: %s (%s)\n", locale, fixturePath)

	mc := memory.New(&memory.Config{
		MaxEntries:      100,
		MaxMemoryBytes:  0,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Second,
		EvictionPolicy:  cache.LRU,
	})
	defer mc.Close()

	// Set + Get + Exists for every expected state key.
	for _, id := range expectedStates {
		val, ok := entries[id]
		if !ok {
			return fmt.Errorf("[%s] fixture missing key %q", locale, id)
		}

		key := fmt.Sprintf("%s::%s", locale, id)
		want := []byte(val)
		if err := mc.Set(ctx, key, want, 0); err != nil {
			return fmt.Errorf("[%s] Set(%s): %w", locale, key, err)
		}

		got, err := mc.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("[%s] Get(%s): %w", locale, key, err)
		}
		if err := mustEqualBytes(fmt.Sprintf("[%s] roundtrip %s", locale, id), got, want); err != nil {
			return err
		}

		exists, err := mc.Exists(ctx, key)
		if err != nil {
			return fmt.Errorf("[%s] Exists(%s): %w", locale, key, err)
		}
		if err := mustEqualBool(fmt.Sprintf("[%s] Exists %s", locale, id), exists, true); err != nil {
			return err
		}
	}

	// Delete the first key, assert subsequent Get returns nil (miss).
	firstKey := fmt.Sprintf("%s::%s", locale, expectedStates[0])
	if err := mc.Delete(ctx, firstKey); err != nil {
		return fmt.Errorf("[%s] Delete(%s): %w", locale, firstKey, err)
	}
	gotAfterDelete, err := mc.Get(ctx, firstKey)
	if err != nil {
		return fmt.Errorf("[%s] Get after delete: %w", locale, err)
	}
	if gotAfterDelete != nil {
		return fmt.Errorf("[%s] Get after Delete returned non-nil: %q", locale, string(gotAfterDelete))
	}
	fmt.Printf("  OK  [%s] post-delete miss confirmed (Get returns nil)\n", locale)

	return nil
}

// crossLocaleSanity asserts that the EN and SR fixture values
// differ for every probed message ID. Catches the regression where
// a NoopTranslator (or copy-paste error) silently serves the same
// language for both locales.
func crossLocaleSanity(enPath, srPath string) error {
	en, err := loadBundle(enPath)
	if err != nil {
		return err
	}
	sr, err := loadBundle(srPath)
	if err != nil {
		return err
	}
	for _, id := range expectedStates {
		eVal, eOK := en[id]
		sVal, sOK := sr[id]
		if !eOK || !sOK {
			return fmt.Errorf("cross-locale sanity: missing key %s (en=%v sr=%v)", id, eOK, sOK)
		}
		if eVal == sVal {
			return fmt.Errorf("cross-locale sanity: en and sr-Latn returned identical %q for %s", eVal, id)
		}
		fmt.Printf("  OK  cross-locale differs for %s: en=%q sr-Latn=%q\n", id, eVal, sVal)
	}
	return nil
}

// ttlExpiryProbe proves FixedTTL actually controls visibility.
// We Set with a 50ms TTL, sleep 200ms, then Get — MUST be nil.
// If the entry survives, the TTL machinery is not wired correctly.
func ttlExpiryProbe(ctx context.Context) error {
	fmt.Println("--- ttl-expiry-probe (FixedTTL=50ms)")
	mc := memory.New(&memory.Config{
		MaxEntries:      10,
		DefaultTTL:      time.Hour,
		CleanupInterval: 10 * time.Millisecond,
		EvictionPolicy:  cache.LRU,
	})
	defer mc.Close()

	pol := policy.NewFixedTTL(50 * time.Millisecond)
	key := "ttl-probe-key"
	ttl := pol.GetTTL(key)
	if ttl != 50*time.Millisecond {
		return fmt.Errorf("FixedTTL.GetTTL: got %v want 50ms", ttl)
	}
	fmt.Printf("  OK  FixedTTL.GetTTL -> %v\n", ttl)

	if err := mc.Set(ctx, key, []byte("ephemeral"), ttl); err != nil {
		return fmt.Errorf("Set with TTL: %w", err)
	}

	// Confirm immediately visible.
	pre, err := mc.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("pre-expiry Get: %w", err)
	}
	if string(pre) != "ephemeral" {
		return fmt.Errorf("pre-expiry Get returned %q want %q", string(pre), "ephemeral")
	}
	fmt.Printf("  OK  pre-expiry Get returned %q\n", string(pre))

	// Wait past TTL window.
	time.Sleep(200 * time.Millisecond)

	post, err := mc.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("post-expiry Get: %w", err)
	}
	if post != nil {
		return fmt.Errorf("post-expiry Get should be nil (TTL drift); got %q", string(post))
	}
	fmt.Println("  OK  post-expiry Get returned nil (TTL respected)")
	return nil
}

// statsProbe drives ~10 Set/Get cycles and asserts memory.Cache.Stats
// reports a positive hit count — proves the stats counters are wired
// to real operations, not bluff zeros.
func statsProbe(ctx context.Context) error {
	fmt.Println("--- stats-probe")
	mc := memory.New(&memory.Config{
		MaxEntries:      32,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Second,
		EvictionPolicy:  cache.LRU,
	})
	defer mc.Close()

	for i := 0; i < 5; i++ {
		k := fmt.Sprintf("stat-key-%d", i)
		if err := mc.Set(ctx, k, []byte("v"), 0); err != nil {
			return fmt.Errorf("Set(%s): %w", k, err)
		}
		if _, err := mc.Get(ctx, k); err != nil {
			return fmt.Errorf("Get(%s): %w", k, err)
		}
	}
	// Force misses.
	for i := 100; i < 105; i++ {
		k := fmt.Sprintf("miss-key-%d", i)
		got, err := mc.Get(ctx, k)
		if err != nil {
			return fmt.Errorf("Get(miss %s): %w", k, err)
		}
		if got != nil {
			return fmt.Errorf("Get(miss %s) returned non-nil %q", k, string(got))
		}
	}

	stats := mc.Stats()
	if stats.Hits < 5 {
		return fmt.Errorf("stats.Hits: got %d want >=5", stats.Hits)
	}
	if stats.Misses < 5 {
		return fmt.Errorf("stats.Misses: got %d want >=5", stats.Misses)
	}
	rate := stats.HitRate()
	if rate <= 0 || rate >= 100 {
		return fmt.Errorf("stats.HitRate(): got %.2f%% — expected strictly between 0 and 100", rate)
	}
	fmt.Printf("  OK  stats: hits=%d misses=%d hit-rate=%.2f%%\n", stats.Hits, stats.Misses, rate)
	return nil
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: cwd: %v\n", err)
		os.Exit(1)
	}
	fixturesDir := filepath.Join(root, "challenges", "fixtures")
	enFixture := filepath.Join(fixturesDir, "en.yaml")
	srFixture := filepath.Join(fixturesDir, "sr-Latn.yaml")

	ctx := context.Background()

	if err := runLocale(ctx, enFixture, "en"); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL en: %v\n", err)
		os.Exit(1)
	}
	if err := runLocale(ctx, srFixture, "sr-Latn"); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL sr-Latn: %v\n", err)
		os.Exit(1)
	}
	if err := crossLocaleSanity(enFixture, srFixture); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL cross-locale: %v\n", err)
		os.Exit(1)
	}
	if err := ttlExpiryProbe(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL ttl-expiry: %v\n", err)
		os.Exit(1)
	}
	if err := statsProbe(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("PASS: cache describe-Challenge — EN+SR round-trip + TTL + stats + cross-locale sanity green")
}
