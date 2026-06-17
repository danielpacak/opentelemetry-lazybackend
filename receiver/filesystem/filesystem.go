// Package filesystem implements a receiver that persists received profiles to
// a directory tree. The layout groups stack traces by container and separates
// them by sample type, e.g.:
//
//	<dir>/<container.id>/<sample-type>/<n>.json
//
// Each JSON file represents a single stack trace (one sample) together with its
// timestamps and attributes.
package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.22.0"
)

const frameTypeAttr = "profile.frame.type"

type Config struct {
	// Dir is the root output directory. Container/sample-type sub-directories
	// are created underneath it on demand.
	Dir string
	// IgnoreProfilesWithoutContainerID skips resource profiles that do not
	// carry a container.id resource attribute.
	IgnoreProfilesWithoutContainerID bool
}

func DefaultConfig() Config {
	return Config{
		Dir: "profiles",
	}
}

type Filesystem struct {
	config Config

	mu       sync.Mutex
	counters map[string]int // dir -> last written file index
}

func NewReceiver(config Config) *Filesystem {
	return &Filesystem{
		config:   config,
		counters: make(map[string]int),
	}
}

// frame is a single entry of a stack trace.
type frame struct {
	Type     string `json:"type"`
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int64  `json:"line,omitempty"`
	Column   int64  `json:"column,omitempty"`
	Address  string `json:"address,omitempty"`
	Mapping  string `json:"mapping,omitempty"`
}

// stackTrace is the JSON payload written for a single sample.
type stackTrace struct {
	ContainerID string            `json:"container_id"`
	ProfileID   string            `json:"profile_id"`
	SampleType  string            `json:"sample_type"`
	Timestamps  []uint64          `json:"timestamps_unix_nano,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Frames      []frame           `json:"frames"`
}

func (f *Filesystem) Receive(ctx context.Context, pd pprofile.Profiles) error {
	return f.consumeProfiles(ctx, pd)
}

func (f *Filesystem) consumeProfiles(_ context.Context, pd pprofile.Profiles) error {
	dict := pd.Dictionary()
	mappingTable := dict.MappingTable()
	locationTable := dict.LocationTable()
	attributeTable := dict.AttributeTable()
	functionTable := dict.FunctionTable()
	stringTable := dict.StringTable()
	stackTable := dict.StackTable()

	rps := pd.ResourceProfiles()
	for i := 0; i < rps.Len(); i++ {
		rp := rps.At(i)

		containerID := "unknown"
		if v, ok := rp.Resource().Attributes().Get(string(semconv.ContainerIDKey)); ok && v.AsString() != "" {
			containerID = v.AsString()
		} else if f.config.IgnoreProfilesWithoutContainerID {
			continue
		}

		sps := rp.ScopeProfiles()
		for j := 0; j < sps.Len(); j++ {
			profiles := sps.At(j).Profiles()
			for k := 0; k < profiles.Len(); k++ {
				profile := profiles.At(k)

				sampleType := stringTable.At(int(profile.SampleType().TypeStrindex()))
				if sampleType == "" {
					sampleType = "unknown"
				}
				profileID := fmt.Sprintf("%x", [16]byte(profile.ProfileID()))

				samples := profile.Samples()
				for l := 0; l < samples.Len(); l++ {
					sample := samples.At(l)

					st := stackTrace{
						ContainerID: containerID,
						ProfileID:   profileID,
						SampleType:  sampleType,
						Attributes:  make(map[string]string),
					}

					for t := 0; t < sample.TimestampsUnixNano().Len(); t++ {
						st.Timestamps = append(st.Timestamps, sample.TimestampsUnixNano().At(t))
					}

					sampleAttrs := sample.AttributeIndices()
					for n := 0; n < sampleAttrs.Len(); n++ {
						attr := attributeTable.At(int(sampleAttrs.At(n)))
						st.Attributes[stringTable.At(int(attr.KeyStrindex()))] = attr.Value().AsString()
					}

					locationIndices := stackTable.At(int(sample.StackIndex())).LocationIndices()
					for m := 0; m < locationIndices.Len(); m++ {
						location := locationTable.At(int(locationIndices.At(m)))

						frameType := frameTypeOf(location, attributeTable, stringTable)

						lines := location.Lines()
						if lines.Len() == 0 {
							file := "<unknown>"
							if location.MappingIndex() > 0 {
								mapping := mappingTable.At(int(location.MappingIndex()))
								file = stringTable.At(int(mapping.FilenameStrindex()))
							}
							st.Frames = append(st.Frames, frame{
								Type:    frameType,
								Address: fmt.Sprintf("%#x", location.Address()),
								Mapping: file,
							})
							continue
						}

						for n := 0; n < lines.Len(); n++ {
							line := lines.At(n)
							function := functionTable.At(int(line.FunctionIndex()))
							st.Frames = append(st.Frames, frame{
								Type:     frameType,
								Function: stringTable.At(int(function.NameStrindex())),
								File:     stringTable.At(int(function.FilenameStrindex())),
								Line:     line.Line(),
								Column:   line.Column(),
							})
						}
					}

					if err := f.write(containerID, sampleType, st); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func frameTypeOf(location pprofile.Location, attributeTable pprofile.KeyValueAndUnitSlice, stringTable pcommon.StringSlice) string {
	attrs := location.AttributeIndices()
	for i := 0; i < attrs.Len(); i++ {
		attr := attributeTable.At(int(attrs.At(i)))
		if stringTable.At(int(attr.KeyStrindex())) == frameTypeAttr {
			return attr.Value().AsString()
		}
	}
	return "unknown"
}

func (f *Filesystem) write(containerID, sampleType string, st stackTrace) error {
	path, err := f.nextPath(containerID, sampleType)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// nextPath ensures <dir>/<container>/<sampleType> exists and returns the path
// of the next <n>.json file in it. Numbering continues after any files already
// present so restarts don't overwrite earlier output.
func (f *Filesystem) nextPath(containerID, sampleType string) (string, error) {
	dir := filepath.Join(f.config.Dir, containerID, sampleType)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	n, ok := f.counters[dir]
	if !ok {
		n = maxIndex(dir)
	}
	n++
	f.counters[dir] = n
	return filepath.Join(dir, strconv.Itoa(n)+".json"), nil
}

// maxIndex returns the highest <n> among <n>.json files in dir, or 0 if none.
func maxIndex(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	max := 0
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		if name == e.Name() {
			continue // not a .json file
		}
		if n, err := strconv.Atoi(name); err == nil && n > max {
			max = n
		}
	}
	return max
}
