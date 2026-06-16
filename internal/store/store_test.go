package store

import "testing"

// TestFTS5RoundTrip verifies the pure-Go SQLite driver was built with FTS5 and
// that index + match works — the core of the offline-search feature.
func TestFTS5RoundTrip(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.Index("incident", "business_rule", "Set assignment group", "if current.priority == 1 ..."); err != nil {
		t.Fatalf("index: %v", err)
	}
	if err := s.Index("change_request", "field", "risk", "Risk of the change"); err != nil {
		t.Fatalf("index: %v", err)
	}

	hits, err := s.Search("assignment")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("want 1 hit, got %d (%+v)", len(hits), hits)
	}
	if hits[0].TableName != "incident" || hits[0].Kind != "business_rule" {
		t.Fatalf("unexpected hit: %+v", hits[0])
	}
}
