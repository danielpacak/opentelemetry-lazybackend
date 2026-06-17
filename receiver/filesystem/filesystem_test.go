package filesystem_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/danielpacak/opentelemetry-lazybackend/receiver/filesystem"
)

func TestReceiveWritesGroupedStackTraces(t *testing.T) {
	dir := t.TempDir()

	r := filesystem.NewReceiver(filesystem.Config{Dir: dir})
	if err := r.Receive(context.Background(), newProfiles()); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	// One samples file and one events file grouped under the container id.
	samplesPath := filepath.Join(dir, "abc123", "samples", "1.json")
	eventsPath := filepath.Join(dir, "abc123", "events", "1.json")

	for _, p := range []string{samplesPath, eventsPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file %s: %v", p, err)
		}
	}

	data, err := os.ReadFile(samplesPath)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if got["container_id"] != "abc123" {
		t.Errorf("container_id = %v, want abc123", got["container_id"])
	}
	if got["sample_type"] != "samples" {
		t.Errorf("sample_type = %v, want samples", got["sample_type"])
	}
	attrs, _ := got["attributes"].(map[string]any)
	if attrs["thread.name"] != "etcd" {
		t.Errorf("attributes[thread.name] = %v, want etcd", attrs["thread.name"])
	}
	frames, _ := got["frames"].([]any)
	if len(frames) != 1 {
		t.Fatalf("frames len = %d, want 1", len(frames))
	}
	frame, _ := frames[0].(map[string]any)
	if frame["function"] != "do_syscall_64" {
		t.Errorf("frame function = %v, want do_syscall_64", frame["function"])
	}
}

// newProfiles builds a minimal Profiles with two resource profiles sharing a
// container id: one CPU "samples" profile and one "events" profile.
func newProfiles() pprofile.Profiles {
	pd := pprofile.NewProfiles()
	dict := pd.Dictionary()

	st := dict.StringTable()
	// Interned strings, referenced by index below.
	st.Append(
		"",                  // 0
		"samples",           // 1
		"events",            // 2
		"thread.name",       // 3
		"do_syscall_64",     // 4
		"sys.c",             // 5
		"profile.frame.type", // 6
		"kernel",            // 7
	)

	// Attribute table: [0] thread.name=etcd, [1] profile.frame.type=kernel.
	threadAttr := dict.AttributeTable().AppendEmpty()
	threadAttr.SetKeyStrindex(3)
	threadAttr.Value().SetStr("etcd")

	frameTypeAttr := dict.AttributeTable().AppendEmpty()
	frameTypeAttr.SetKeyStrindex(6)
	frameTypeAttr.Value().SetStr("kernel")

	// Function "do_syscall_64" in "sys.c".
	fn := dict.FunctionTable().AppendEmpty()
	fn.SetNameStrindex(4)
	fn.SetFilenameStrindex(5)

	// Location referencing the function, tagged as a kernel frame.
	loc := dict.LocationTable().AppendEmpty()
	loc.AttributeIndices().Append(1)
	line := loc.Lines().AppendEmpty()
	line.SetFunctionIndex(0)
	line.SetLine(42)

	// Stack referencing the single location.
	stack := dict.StackTable().AppendEmpty()
	stack.LocationIndices().Append(0)

	addProfile(pd, "abc123", 1 /* samples */)
	addProfile(pd, "abc123", 2 /* events */)
	return pd
}

func addProfile(pd pprofile.Profiles, containerID string, sampleTypeStrindex int32) {
	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("container.id", containerID)

	profile := rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	profile.SampleType().SetTypeStrindex(sampleTypeStrindex)

	sample := profile.Samples().AppendEmpty()
	sample.SetStackIndex(0)
	sample.AttributeIndices().Append(0) // thread.name=etcd
	sample.TimestampsUnixNano().Append(1758603335517877230)
}
