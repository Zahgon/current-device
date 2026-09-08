package device_test

import (
	"encoding/json"
	"os"
	"testing"

	device "github.com/matthewhudson/current-device-go"
)

// oracleCase is one scenario replayed against the original TypeScript
// implementation (src/index.ts executed under Node) and recorded verbatim.
// Regenerate with `make oracle`. This is the migration's equivalence proof.
type oracleCase struct {
	Name string `json:"name"`
	Env  struct {
		UserAgent              string  `json:"userAgent"`
		Platform               string  `json:"platform"`
		MaxTouchPoints         int     `json:"maxTouchPoints"`
		HasScreenOrientation   bool    `json:"hasScreenOrientation"`
		ScreenOrientationType  string  `json:"screenOrientationType"`
		HasOnOrientationChange bool    `json:"hasOnOrientationChange"`
		HasWindowOrientation   bool    `json:"hasWindowOrientation"`
		WindowOrientation      int     `json:"windowOrientation"`
		InnerWidth             float64 `json:"innerWidth"`
		InnerHeight            float64 `json:"innerHeight"`
		HasCordova             bool    `json:"hasCordova"`
		LocationProtocol       string  `json:"locationProtocol"`
		HasProcessObject       bool    `json:"hasProcessObject"`
	} `json:"env"`
	InitialClassName string          `json:"initialClassName"`
	OS               string          `json:"os"`
	Type             string          `json:"type"`
	Orientation      string          `json:"orientation"`
	ClassName        string          `json:"className"`
	EventName        string          `json:"eventName"`
	Methods          map[string]bool `json:"methods"`
}

func loadOracle(t *testing.T) []oracleCase {
	t.Helper()

	raw, err := os.ReadFile("testdata/typescript_oracle.json")
	if err != nil {
		t.Fatalf("read oracle: %v", err)
	}

	var cases []oracleCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("decode oracle: %v", err)
	}

	if len(cases) == 0 {
		t.Fatal("oracle is empty")
	}

	return cases
}

func (c oracleCase) env() device.Env {
	return device.Env{
		UserAgent:              c.Env.UserAgent,
		Platform:               c.Env.Platform,
		MaxTouchPoints:         c.Env.MaxTouchPoints,
		HasScreenOrientation:   c.Env.HasScreenOrientation,
		ScreenOrientationType:  c.Env.ScreenOrientationType,
		HasOnOrientationChange: c.Env.HasOnOrientationChange,
		HasWindowOrientation:   c.Env.HasWindowOrientation,
		WindowOrientation:      c.Env.WindowOrientation,
		InnerWidth:             c.Env.InnerWidth,
		InnerHeight:            c.Env.InnerHeight,
		HasCordova:             c.Env.HasCordova,
		LocationProtocol:       c.Env.LocationProtocol,
		HasProcessObject:       c.Env.HasProcessObject,
	}
}

// TestDifferentialAgainstTypeScript replays every recorded scenario through the
// Go port and requires byte-identical results for the cached properties, the
// resulting <html> class string, the orientation event name, and all 27
// predicates.
func TestDifferentialAgainstTypeScript(t *testing.T) {
	for _, tc := range loadOracle(t) {
		t.Run(tc.Name, func(t *testing.T) {
			d := device.New(tc.env())

			// Mirrors the TypeScript module's import-time side effects: the
			// detection cascade runs first, then handleOrientation().
			className := d.ClassNames(tc.InitialClassName)
			className = d.HandleOrientation(className)

			if got := string(d.OS()); got != tc.OS {
				t.Errorf("os = %q, want %q", got, tc.OS)
			}

			if got := string(d.Type()); got != tc.Type {
				t.Errorf("type = %q, want %q", got, tc.Type)
			}

			if got := string(d.Orientation()); got != tc.Orientation {
				t.Errorf("orientation = %q, want %q", got, tc.Orientation)
			}

			if className != tc.ClassName {
				t.Errorf("className = %q, want %q", className, tc.ClassName)
			}

			if got := d.OrientationEventName(); got != tc.EventName {
				t.Errorf("orientation event = %q, want %q", got, tc.EventName)
			}

			for name, want := range tc.Methods {
				predicate, ok := d.Predicate(name)
				if !ok {
					t.Fatalf("predicate %q is missing from the Go port", name)
				}

				if got := predicate(); got != want {
					t.Errorf("%s() = %v, want %v", name, got, want)
				}
			}
		})
	}
}

// TestOracleCoversEveryPredicate guards against the oracle silently losing a
// method, which would let a regression pass unnoticed.
func TestOracleCoversEveryPredicate(t *testing.T) {
	cases := loadOracle(t)

	for _, name := range device.PredicateNames() {
		if _, ok := cases[0].Methods[name]; !ok {
			t.Errorf("oracle does not record predicate %q", name)
		}
	}
}

// TestOracleExercisesBothOutcomes proves the corpus is not trivially satisfied
// by a port that hardcodes false: every predicate must be observed true and
// false at least once across the recorded scenarios.
func TestOracleExercisesBothOutcomes(t *testing.T) {
	cases := loadOracle(t)
	seenTrue, seenFalse := map[string]bool{}, map[string]bool{}

	for _, tc := range cases {
		for name, value := range tc.Methods {
			if value {
				seenTrue[name] = true
			} else {
				seenFalse[name] = true
			}
		}
	}

	for _, name := range device.PredicateNames() {
		if !seenTrue[name] {
			t.Errorf("no recorded scenario where %s() is true", name)
		}

		if !seenFalse[name] {
			t.Errorf("no recorded scenario where %s() is false", name)
		}
	}
}
