package skilldoc

// Fence topology: which GENERATED fence carries which commands of a group.
//
// A fence id is either the bare group name ("channel") — the group's
// catch-all fence — or the group plus a bracketed verb-prefix claim list
// ("channel[silence-rule,inhibit-rule]") — a subset fence that claims every
// verb starting with one of the prefixes. A group's fences may live in
// different cards; together they must cover the group exactly: every verb
// lands in exactly one fence, each prefix claims at least one verb, and any
// unclaimed remainder requires the catch-all fence to exist.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// FenceSpec is one parsed fence id.
type FenceSpec struct {
	Group    string
	Prefixes []string // empty → the group's catch-all fence
}

// ID renders the spec back to its marker id ("group" or "group[p1,p2]").
func (s FenceSpec) ID() string {
	if len(s.Prefixes) == 0 {
		return s.Group
	}
	return s.Group + "[" + strings.Join(s.Prefixes, ",") + "]"
}

// fenceIDRe accepts "group" or "group[prefix,prefix,...]". Group and prefix
// share the verb charset; no spaces, so a malformed claim list fails loudly
// instead of silently truncating at the first space.
var fenceIDRe = regexp.MustCompile(`^([a-z0-9-]+)(?:\[([a-z0-9-]+(?:,[a-z0-9-]+)*)\])?$`)

// ParseFenceID parses a fence id as found in a GENERATED marker.
func ParseFenceID(id string) (FenceSpec, error) {
	m := fenceIDRe.FindStringSubmatch(id)
	if m == nil {
		return FenceSpec{}, fmt.Errorf("malformed fence id %q (want group or group[verb-prefix,…])", id)
	}
	spec := FenceSpec{Group: m[1]}
	if m[2] != "" {
		spec.Prefixes = strings.Split(m[2], ",")
	}
	return spec, nil
}

// FenceLoc is one GENERATED start marker found in a doc body.
type FenceLoc struct {
	ID     string
	Offset int // byte offset of the start marker
}

// fenceStartRe matches a start marker and captures its fence id; the literal
// " START " cannot appear in an end marker, so ends never match.
var fenceStartRe = regexp.MustCompile(`<!-- GENERATED:([^ ]+) START `)

// FenceLocs returns every GENERATED start marker in body, in document order.
func FenceLocs(body string) []FenceLoc {
	var locs []FenceLoc
	for _, m := range fenceStartRe.FindAllStringSubmatchIndex(body, -1) {
		locs = append(locs, FenceLoc{ID: body[m[2]:m[3]], Offset: m[0]})
	}
	return locs
}

// FindFence locates the full fenced block for id in body: start is the byte
// offset of the start marker, end is the offset just past the end marker.
// ok is false when either marker is missing.
func FindFence(body, id string) (start, end int, ok bool) {
	si := strings.Index(body, FenceStart(id))
	if si < 0 {
		return 0, 0, false
	}
	endMarker := FenceEnd(id)
	ei := strings.Index(body[si:], endMarker)
	if ei < 0 {
		return 0, 0, false
	}
	return si, si + ei + len(endMarker), true
}

// matchesPrefix reports whether verb falls under the claim prefix p: the verb
// IS p, or continues past it at a hyphen boundary — so "rule" claims
// "rule-create" but never "rule2-list". An unbounded prefix match would let a
// near-miss verb join the wrong card with a clean single-owner partition that
// no topology check could flag.
func matchesPrefix(verb, p string) bool {
	return verb == p || strings.HasPrefix(verb, p+"-")
}

// RenderGroupFences renders the fenced block for every fence of one command
// group. ids must be the complete set of fence ids that exist for the group
// across all cards — the catch-all fence renders whatever its sibling subset
// fences leave unclaimed, so a fence cannot be rendered in isolation.
// Topology problems (a verb claimed twice, a prefix claiming nothing, verbs
// left over with no catch-all, a duplicated id) come back as violations;
// rendered blocks are still returned for the fences that parsed.
func RenderGroupFences(d Dump, group string, ids []string) (map[string]string, []string) {
	var violations []string
	var specs []FenceSpec
	seen := map[string]bool{}
	for _, id := range ids {
		spec, err := ParseFenceID(id)
		if err != nil {
			violations = append(violations, err.Error())
			continue
		}
		if spec.Group != group {
			violations = append(violations, fmt.Sprintf("fence %q does not belong to group %q", id, group))
			continue
		}
		if seen[spec.ID()] {
			violations = append(violations, fmt.Sprintf("fence %q appears more than once", id))
			continue
		}
		seen[spec.ID()] = true
		specs = append(specs, spec)
	}

	byID := make(map[string][]Command, len(specs))
	catchAll := ""
	for _, s := range specs {
		if len(s.Prefixes) == 0 {
			catchAll = s.ID()
		}
	}

	// Partition the group's verbs among the specs. A verb matching several
	// prefixes of ONE spec is fine (all count as live); matching prefixes of
	// two specs is a double claim.
	prefixHit := map[string]bool{} // spec id + "\x00" + prefix → claimed something
	var unclaimed []string
	for _, c := range groupCommands(d, group) {
		verb := verbOf(c.Path)
		var owners []string
		for _, s := range specs {
			matched := false
			for _, p := range s.Prefixes {
				if matchesPrefix(verb, p) {
					prefixHit[s.ID()+"\x00"+p] = true
					matched = true
				}
			}
			if matched {
				owners = append(owners, s.ID())
			}
		}
		switch {
		case len(owners) > 1:
			violations = append(violations, fmt.Sprintf("verb %q claimed by %s", verb, strings.Join(owners, " and ")))
		case len(owners) == 1:
			byID[owners[0]] = append(byID[owners[0]], c)
		case catchAll != "":
			byID[catchAll] = append(byID[catchAll], c)
		default:
			unclaimed = append(unclaimed, verb)
		}
	}
	if len(unclaimed) > 0 {
		violations = append(violations, fmt.Sprintf("group %q has no catch-all fence for unclaimed verbs: %s", group, strings.Join(unclaimed, ", ")))
	}
	for _, s := range specs {
		for _, p := range s.Prefixes {
			if !prefixHit[s.ID()+"\x00"+p] {
				violations = append(violations, fmt.Sprintf("fence %q: prefix %q claims no verb", s.ID(), p))
			}
		}
	}

	out := make(map[string]string, len(specs))
	for _, s := range specs {
		out[s.ID()] = renderFence(s.ID(), byID[s.ID()])
	}
	sort.Strings(violations)
	return out, violations
}
