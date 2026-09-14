package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

// The endpoint returns the data source's native Prometheus payload, not the
// Flashduty {request_id, error, data} envelope gfStub always sends, so this
// test wires its own stub server instead of using gfStub.
func newPrometheusLabelValuesStub(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}
}

func TestMonitPrometheusLabelValuesSendsPathAndHeader(t *testing.T) {
	saveAndResetGlobals(t)

	var gotMethod, gotPath, gotDSID string
	newPrometheusLabelValuesStub(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotDSID = r.Header.Get("X-DSID")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"success","data":["api","db","worker"]}`)
	})

	out, err := execCommand(
		"monit", "prometheus-api-v1-label-{label_name}-values", "job",
		"--data-source-id", "12345",
		"--json",
	)
	if err != nil {
		t.Fatalf("[monit-prometheus-label-values] unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("[monit-prometheus-label-values] method = %q", gotMethod)
	}
	if gotPath != "/monit/prometheus/api/v1/label/job/values" {
		t.Fatalf("[monit-prometheus-label-values] path = %q", gotPath)
	}
	if gotDSID != "12345" {
		t.Fatalf("[monit-prometheus-label-values] X-DSID = %q", gotDSID)
	}
	if !strings.Contains(out, `"db"`) {
		t.Fatalf("[monit-prometheus-label-values] output = %s", out)
	}
}

func TestMonitPrometheusLabelValuesRequiresDataSourceID(t *testing.T) {
	saveAndResetGlobals(t)
	newPrometheusLabelValuesStub(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not be sent")
	})

	_, err := execCommand("monit", "prometheus-api-v1-label-{label_name}-values", "job")
	if err == nil {
		t.Fatal("[monit-prometheus-label-values-missing-flag] expected an error")
	}
	if !strings.Contains(err.Error(), "data-source-id") {
		t.Fatalf("[monit-prometheus-label-values-missing-flag] err = %v", err)
	}
}

func TestMonitPrometheusLabelValuesSurfacesDataSourceError(t *testing.T) {
	saveAndResetGlobals(t)
	newPrometheusLabelValuesStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"status":"error","errorType":"bad_data","error":"unknown label name"}`)
	})

	_, err := execCommand(
		"monit", "prometheus-api-v1-label-{label_name}-values", "job",
		"--data-source-id", "12345",
	)
	if err == nil {
		t.Fatal("[monit-prometheus-label-values-ds-error] expected an error")
	}
	if !strings.Contains(err.Error(), "unknown label name") {
		t.Fatalf("[monit-prometheus-label-values-ds-error] err = %v", err)
	}
}
