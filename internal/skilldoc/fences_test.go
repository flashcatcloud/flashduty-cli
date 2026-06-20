package skilldoc

import (
	"strings"
	"testing"
)

func TestCheckFences_DetectsStale(t *testing.T) {
	d := generatorDump()
	fresh := GenerateFence(d, "status-page")

	freshDoc := "# Status page\n\nintro\n\n" + fresh + "\n\nfooter\n"
	staleDoc := "# Status page\n\n" +
		FenceStart("status-page") + "\n\n### change-create\nWRONG STALE CONTENT\n\n" +
		FenceEnd("status-page") + "\n"
	noFenceDoc := "# Status page\n\nJust prose, no generated fence at all.\n"

	docs := []Doc{
		{Path: "fresh", Body: freshDoc},
		{Path: "stale", Body: staleDoc},
		{Path: "none", Body: noFenceDoc},
	}
	issues := CheckFences(d, docs)

	byDoc := map[string][]Issue{}
	for _, is := range issues {
		byDoc[is.Doc] = append(byDoc[is.Doc], is)
	}
	if n := len(byDoc["fresh"]); n != 0 {
		t.Errorf("fresh doc: want 0 issues, got %d: %+v", n, byDoc["fresh"])
	}
	if n := len(byDoc["stale"]); n != 1 || byDoc["stale"][0].Kind != "stale-fence" {
		t.Errorf("stale doc: want 1 stale-fence, got %+v", byDoc["stale"])
	}
	if n := len(byDoc["none"]); n != 0 {
		t.Errorf("no-fence doc: want 0 issues, got %d: %+v", n, byDoc["none"])
	}
}

func TestCheckFences_MalformedFenceIsStale(t *testing.T) {
	d := generatorDump()
	// Start marker without a matching end marker.
	doc := Doc{Path: "broken", Body: FenceStart("status-page") + "\n### change-create\n(no end marker)\n"}
	issues := CheckFences(d, []Doc{doc})
	if len(issues) != 1 || issues[0].Kind != "stale-fence" {
		t.Errorf("malformed fence should be stale-fence: %+v", issues)
	}
	if !strings.Contains(issues[0].Detail, "status-page") {
		t.Errorf("detail should name the group: %+v", issues)
	}
}
