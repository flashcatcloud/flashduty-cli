package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRawJSONIsBodyOnly(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[json.RawMessage](), reflect.TypeFor[*json.RawMessage]()} {
		if kind, scalar := scalarKind(typ); scalar {
			t.Fatalf("JSON params became %s flag", kind)
		}
	}
	if kind, scalar := scalarKind(reflect.TypeFor[[]int64]()); !scalar || kind != "[]int" {
		t.Fatalf("ordinary integer arrays changed: %s %v", kind, scalar)
	}
}
