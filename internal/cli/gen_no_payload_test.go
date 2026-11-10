package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

// newRawStub points the client at a server that answers every request with the
// given status, headers and body bytes, bypassing newGFStub's envelope. Tests
// below use it for the two things an envelope stub cannot produce: a success
// body that is not envelope JSON (which the SDK surfaces on Response.Raw), and
// a 200 that reports no request id at all.
func newRawStub(t *testing.T, contentType string, body []byte, setRequestID bool) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if setRequestID {
			w.Header().Set("Flashcat-Request-Id", gfStubRequestID)
		}
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(server.URL))
	}
}

// TestGeneratedNoPayloadLineIsTheWholeDefaultOutput pins the default output of a
// generated command whose endpoint returns no payload. Equality on the whole
// output, not a substring, is the contract: callers read this through `head`,
// so a line pushed down by anything printed above it is a regression a
// strings.Contains assertion would pass.
//
// What it must NOT be is the previous output, "OK: POST /channel/disable" — an
// echo of the request, which restated the path the caller had just typed and
// so read as a description of a result nobody had fetched.
func TestGeneratedNoPayloadLineIsTheWholeDefaultOutput(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	stdout, stderr, err := execCommandSplit("channel", "disable", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimRight(stdout, "\n"); got != gfStubNoPayloadLine {
		t.Fatalf("default output must be exactly %q, got:\n%s", gfStubNoPayloadLine, stdout)
	}
	if stderr != "" {
		t.Errorf("a plain write must print nothing on stderr, got:\n%s", stderr)
	}
}

// TestGeneratedNoPayloadStructuredCarriesOnlyTheRequestID guards the machine
// path. The old shape was {"message": "OK: POST /channel/disable"} — a rendered
// English sentence in a JSON field, which a caller could only string-match. The
// new shape carries the request id as data, and carries nothing else, because
// nothing else is known: the endpoint's 200 envelope declares `data` as an
// empty object.
//
// The TOON subtest is not duplicate coverage. TOON is a different marshaller
// (toon.Marshal, vs json.MarshalIndent over output.NullUnsetInstants), so a
// change that broke only the TOON path would pass every JSON assertion.
func TestGeneratedNoPayloadStructuredCarriesOnlyTheRequestID(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		saveAndResetGlobals(t)
		newGFStub(t)

		stdout, _, err := execCommandSplit("channel", "disable", "1", "--json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var ack map[string]any
		if err := json.Unmarshal([]byte(stdout), &ack); err != nil {
			t.Fatalf("output is not a JSON object (%v):\n%s", err, stdout)
		}
		if got := ack["request_id"]; got != gfStubRequestID {
			t.Errorf("request_id = %v, want %q", got, gfStubRequestID)
		}
		if len(ack) != 1 {
			t.Errorf("structured acknowledgement must carry request_id alone, got %v", ack)
		}
	})

	t.Run("toon", func(t *testing.T) {
		saveAndResetGlobals(t)
		newGFStub(t)

		stdout, _, err := execCommandSplit("channel", "disable", "1", "--output-format", "toon")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout, "request_id") || !strings.Contains(stdout, gfStubRequestID) {
			t.Errorf("toon output missing the request id:\n%s", stdout)
		}
		if strings.Contains(stdout, "OK:") {
			t.Errorf("toon output must not carry the table-mode sentence:\n%s", stdout)
		}
	})
}

// TestGeneratedNoPayloadOmitsAnAbsentRequestID covers the case the two wordings
// exist for. A 200 with an empty body never parses an envelope, so the id can
// only come from the Flashcat-Request-Id header; with neither, there is no id.
// The line must then say so rather than print a hole, "(request_id )", and the
// structured form must emit an empty object rather than invent request_id: "".
func TestGeneratedNoPayloadOmitsAnAbsentRequestID(t *testing.T) {
	t.Run("table", func(t *testing.T) {
		saveAndResetGlobals(t)
		newRawStub(t, "", nil, false)

		stdout, _, err := execCommandSplit("channel", "disable", "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimRight(stdout, "\n"); got != noPayloadLine {
			t.Fatalf("output must be exactly %q, got:\n%s", noPayloadLine, stdout)
		}
	})

	t.Run("json", func(t *testing.T) {
		saveAndResetGlobals(t)
		newRawStub(t, "", nil, false)

		stdout, _, err := execCommandSplit("channel", "disable", "1", "--json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var ack map[string]any
		if err := json.Unmarshal([]byte(stdout), &ack); err != nil {
			t.Fatalf("output is not a JSON object (%v):\n%s", err, stdout)
		}
		if len(ack) != 0 {
			t.Errorf("an unknown request id must be absent, not empty; got %v", ack)
		}
	})

	// TOON marshals an all-omitempty struct through its own code path, so what
	// it prints for a record with nothing in it does not follow from the JSON
	// case above. Whatever it is, it must not claim an id: a literal
	// "request_id" with nothing after it would read as a server that answered
	// with a blank one.
	t.Run("toon", func(t *testing.T) {
		saveAndResetGlobals(t)
		newRawStub(t, "", nil, false)

		stdout, _, err := execCommandSplit("channel", "disable", "1", "--output-format", "toon")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(stdout, "request_id") {
			t.Errorf("toon output names a request id the server never sent:\n%q", stdout)
		}
	})
}

// exportCSV is the body newRawStub serves for the download tests below. Its
// bytes are deliberately not valid JSON, which is what makes the SDK surface
// them on Response.Raw.
const exportCSV = "issue_id,error_type\nabc,Error\n"

// issueExportArgs drives the generated `rum issue-export` command, one of the
// eight endpoints whose 200 declares no application/json body at all. The
// window goes through --data because both bounds are required epoch
// milliseconds.
var issueExportArgs = []string{
	"rum", "issue-export",
	"--data", `{"start_time":1772611200000,"end_time":1775961914595}`,
}

// TestGeneratedDownloadStreamsRawBodyInsteadOfTheNoPayloadLine is the other
// half of the branch printRawOrNoPayload chooses between. Response.Raw is
// populated only for a success body that is not envelope JSON, so that guard —
// not anything in the spec read at generation time — is what tells a CSV
// download apart from a mutation at runtime. The bytes must reach stdout
// verbatim, with no acknowledgement line wrapped around them, or `> file.csv`
// writes a corrupt file.
func TestGeneratedDownloadStreamsRawBodyInsteadOfTheNoPayloadLine(t *testing.T) {
	saveAndResetGlobals(t)
	newRawStub(t, "text/csv", []byte(exportCSV), true)

	stdout, _, err := execCommandSplit(issueExportArgs...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != exportCSV {
		t.Fatalf("CSV body must reach stdout verbatim:\nwant: %q\n got: %q", exportCSV, stdout)
	}
}

// TestGeneratedDownloadReportsNoPayloadWhenTheServerSendsNoBytes is the same command
// answered with an empty body. There is nothing to stream, so it falls through
// to the acknowledgement — and "the API returned no payload" is the literal
// truth of a zero-byte download, which is why one wording serves both the
// mutations and this case instead of two.
func TestGeneratedDownloadReportsNoPayloadWhenTheServerSendsNoBytes(t *testing.T) {
	saveAndResetGlobals(t)
	newRawStub(t, "text/csv", nil, true)

	stdout, _, err := execCommandSplit(issueExportArgs...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimRight(stdout, "\n"); got != gfStubNoPayloadLine {
		t.Fatalf("output must be exactly %q, got:\n%s", gfStubNoPayloadLine, stdout)
	}
}

// cannedPathEcho matches the output this change removed: a literal
// acknowledgement that names an HTTP verb and a request path.
var cannedPathEcho = regexp.MustCompile(`"OK: (GET|POST|PUT|PATCH|DELETE) /`)

// TestNoCommandEchoesItsRequestPath is the regression guard on the committed
// artifact rather than on the generator: the generated files are derived, so a
// generator change that is not regenerated would leave these untouched. It
// covers the curated commands too, because three of them carried the same
// canned line and a curated command WINS over its generated twin at
// registration (genAddLeaf keeps the first name it sees), so fixing only the
// generator would have left the command users actually reach unfixed.
func TestNoCommandEchoesItsRequestPath(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if loc := cannedPathEcho.FindIndex(src); loc != nil {
			t.Errorf("%s still acknowledges a write by echoing its request path (%q) — "+
				"report what the API returned, or say it returned nothing (printRawOrNoPayload)",
				name, src[loc[0]:loc[1]])
		}
	}
}
