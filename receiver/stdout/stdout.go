package stdout

import (
	"context"
	"fmt"
	"slices"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

type Config struct {
	ExportResourceAttributes         bool
	ExportProfileAttributes          bool
	ExportSampleAttributes           bool
	ExportStackFrames                bool
	ExportStackFrameTypes            []string
	IgnoreProfilesWithoutContainerID bool
}

func DefaultConfig() Config {
	return Config{
		ExportResourceAttributes: true,
		ExportProfileAttributes:  true,
		ExportSampleAttributes:   true,
		ExportStackFrames:        true,
	}
}

type Stdout struct {
	config Config
}

func NewReceiver(config Config) *Stdout {
	return &Stdout{
		config: config,
	}
}

func (s *Stdout) Receive(ctx context.Context, pd pprofile.Profiles) error {
	return s.consumeProfiles(ctx, pd)
}

func (s *Stdout) consumeProfiles(_ context.Context, pd pprofile.Profiles) error {
	mappingTable := pd.ProfilesDictionary().MappingTable()
	locationTable := pd.ProfilesDictionary().LocationTable()
	attributeTable := pd.ProfilesDictionary().AttributeTable()
	functionTable := pd.ProfilesDictionary().FunctionTable()
	stringTable := pd.ProfilesDictionary().StringTable()

	rps := pd.ResourceProfiles()
	for i := 0; i < rps.Len(); i++ {
		rp := rps.At(i)

		if s.config.IgnoreProfilesWithoutContainerID {
			containerID, ok := rp.Resource().Attributes().Get("container.id")
			if !ok || containerID.AsString() == "" {
				fmt.Println("--------------- New Resource Profile --------------")
				fmt.Println("              SKIPPED (no container.id)")
				fmt.Printf("-------------- End Resource Profile ---------------\n\n")
				continue
			}
		}

		fmt.Println("--------------- New Resource Profile --------------")
		if s.config.ExportResourceAttributes {
			if rp.Resource().Attributes().Len() > 0 {
				rp.Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
					fmt.Printf("  %s: %v\n", k, v.AsString())
					return true
				})
			}
		}

		sps := rp.ScopeProfiles()
		for j := 0; j < sps.Len(); j++ {
			pcs := sps.At(j).Profiles()
			for k := 0; k < pcs.Len(); k++ {
				profile := pcs.At(k)

				fmt.Println("------------------- New Profile -------------------")
				fmt.Printf("  ProfileID: %x\n", [16]byte(profile.ProfileID()))
				fmt.Printf("  Time: %v\n", profile.Time().AsTime())
				fmt.Printf("  Duration: %v\n", profile.Duration())
				fmt.Printf("  PeriodType: [%v, %v, %v]\n",
					stringTable.At(int(profile.PeriodType().TypeStrindex())),
					stringTable.At(int(profile.PeriodType().UnitStrindex())),
					profile.PeriodType().AggregationTemporality().String())
				fmt.Printf("  Period: %v\n", profile.Period())
				fmt.Printf("  Dropped attributes count: %d\n", profile.DroppedAttributesCount())

				sampleType := "samples"
				for n := 0; n < profile.SampleType().Len(); n++ {
					sampleType = stringTable.At(int(profile.SampleType().At(n).TypeStrindex()))
					fmt.Printf("  SampleType: %s\n", sampleType)
				}
				profileAttrs := profile.AttributeIndices()
				if profileAttrs.Len() > 0 {
					for n := 0; n < profileAttrs.Len(); n++ {
						attr := attributeTable.At(int(profileAttrs.At(n)))
						fmt.Printf("  %s: %s\n", attr.Key(), attr.Value().AsString())
					}
					fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
				}

				samples := profile.Sample()

				for l := 0; l < samples.Len(); l++ {
					sample := samples.At(l)

					fmt.Println("------------------- New Sample --------------------")

					for t := 0; t < sample.TimestampsUnixNano().Len(); t++ {
						sampleTimestampUnixNano := sample.TimestampsUnixNano().At(t)
						sampleTimestampNano := time.Unix(0, int64(sampleTimestampUnixNano))
						fmt.Printf("  Timestamp[%d]: %d (%s)\n", t,
							sampleTimestampUnixNano,
							sampleTimestampNano)
					}

					if s.config.ExportSampleAttributes {
						sampleAttrs := sample.AttributeIndices()
						for n := 0; n < sampleAttrs.Len(); n++ {
							attr := attributeTable.At(int(sampleAttrs.At(n)))
							fmt.Printf("  %s: %s\n", attr.Key(), attr.Value().AsString())
						}
						fmt.Println("---------------------------------------------------")
					}

					profileLocationsIndices := profile.LocationIndices()

					if s.config.ExportStackFrames {
						for m := sample.LocationsStartIndex(); m < sample.LocationsStartIndex()+sample.LocationsLength(); m++ {
							location := locationTable.At(int(profileLocationsIndices.At(int(m))))
							locationAttrs := location.AttributeIndices()

							unwindType := "unknown"
							for la := 0; la < locationAttrs.Len(); la++ {
								attr := attributeTable.At(int(locationAttrs.At(la)))
								if attr.Key() == "profile.frame.type" {
									unwindType = attr.Value().AsString()
									break
								}
							}

							if len(s.config.ExportStackFrameTypes) > 0 &&
								!slices.Contains(s.config.ExportStackFrameTypes, unwindType) {
								continue
							}

							locationLine := location.Line()
							if locationLine.Len() == 0 {
								filename := "<unknown>"
								if location.HasMappingIndex() {
									mapping := mappingTable.At(int(location.MappingIndex()))
									filename = stringTable.At(int(mapping.FilenameStrindex()))
								}
								fmt.Printf("Instrumentation: %s: Function: %#04x, File: %s\n", unwindType, location.Address(), filename)
							}

							for n := 0; n < locationLine.Len(); n++ {
								line := locationLine.At(n)
								function := functionTable.At(int(line.FunctionIndex()))
								functionName := stringTable.At(int(function.NameStrindex()))
								fileName := stringTable.At(int(function.FilenameStrindex()))
								fmt.Printf("Instrumentation: %s, Function: %s, File: %s, Line: %d, Column: %d\n",
									unwindType, functionName, fileName, line.Line(), line.Column())
							}
						}
					}

					fmt.Println("------------------- End Sample --------------------")
				}
				fmt.Println("------------------- End Profile -------------------")
			}
		}

		fmt.Printf("-------------- End Resource Profile ---------------\n\n")
	}
	return nil
}
