package skilldoc

import (
	"os"
	"strings"
	"testing"
)

func TestChannelCardDisambiguatesFlashcatWorkspace(t *testing.T) {
	body, err := os.ReadFile("../../skills/flashduty/reference/channel.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"协作空间",
		"Flashcat workspace",
		"灭火图",
		"firemap",
		"do not silently switch",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("channel card must disambiguate Flashduty channel vs Flashcat workspace; missing %q", want)
		}
	}
}
