# OpenTelemetry Profiles Lazy Backend

``` mermaid
flowchart LR
  opentelemetry-ebpf-profiler
  opentelemetry-profiles-lazybackend
  clickhouse@{ shape: cyl, label: "ClickHouse" }
  sqlite@{ shape: cyl, label: "SQLite" }
  stdout["Stdout"]

  opentelemetry-ebpf-profiler e1@--> opentelemetry-profiles-lazybackend
  opentelemetry-profiles-lazybackend e2@--> clickhouse
  opentelemetry-profiles-lazybackend e3@--> sqlite
  opentelemetry-profiles-lazybackend e4@--> stdout

  e1@{ animate: true }
  e2@{ animate: true }
  e3@{ animate: true }
  e4@{ animate: true }
```

```
git clone git@github.com:open-telemetry/opentelemetry-ebpf-profiler.git
cd opentelemetry-ebpf-profiler
```

```
make agent
```

:coffee: :coffee: :coffee:

``` console
$ sudo ./ebpf-profiler -collection-agent="localhost:4137" -disable-tls
INFO[0000] Starting OTEL profiling agent v0.0.0 (revision main-69066441, build timestamp 1758215582) 
INFO[0000] Interpreter tracers: perl,php,python,hotspot,ruby,v8,dotnet,go,labels 
INFO[0000] Found offsets: task stack 0x20, pt_regs 0x3f58, tpbase 0x2468 
INFO[0000] Supports generic eBPF map batch operations   
INFO[0000] Supports LPM trie eBPF map batch operations  
INFO[0000] eBPF tracer loaded                           
INFO[0000] Attached tracer program                      
INFO[0000] Attached sched monitor                       
```

```
git@github.com:danielpacak/opentelemetry-profiles-lazybackend.git
cd opentelemetry-profiles-lazybackend
```

```
go build
```

``` console
$ ./opentelemetry-profiles-lazybackend
2025/09/19 11:45:49 INFO starting profiles lazy backend server pid=14049 uid=1000 gid=1000
--------------- New Resource Profile --------------
  container.id: 
------------------- New Profile -------------------
  ProfileID: 00000000000000000000000000000000
  Dropped attributes count: 0
  SampleType: samples
------------------- New Sample --------------------
  thread.name: ebpf-profiler
  process.executable.name: ebpf-profiler
  process.executable.path: /home/dpacak/go/src/github.com/danielpacak/opentelemetry-ebpf-profiler/ebpf-profiler
  process.pid: 14168
  thread.id: 14176
---------------------------------------------------
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/nativeunwind/elfunwindinfo.(*elfExtractor).parseFDE, File: /agent/nativeunwind/elfunwindinfo/elfehframe.go, Line: 1004, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/nativeunwind/elfunwindinfo.(*elfExtractor).walkBinSearchTable, File: /agent/nativeunwind/elfunwindinfo/elfehframe.go, Line: 1183, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/nativeunwind/elfunwindinfo.(*elfExtractor).parseEHFrame, File: /agent/nativeunwind/elfunwindinfo/elfehframe.go, Line: 1238, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/nativeunwind/elfunwindinfo.extractFile, File: /agent/nativeunwind/elfunwindinfo/stackdeltaextraction.go, Line: 207, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/nativeunwind/elfunwindinfo.ExtractELF, File: /agent/nativeunwind/elfunwindinfo/stackdeltaextraction.go, Line: 180, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/nativeunwind/elfunwindinfo.(*ELFStackDeltaProvider).GetIntervalStructuresForFile, File: /agent/nativeunwind/elfunwindinfo/stackdeltaprovider.go, Line: 37, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/processmanager/execinfomanager.(*ExecutableInfoManager).AddOrIncRef, File: /agent/processmanager/execinfomanager/manager.go, Line: 187, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/processmanager.(*ProcessManager).handleNewMapping, File: /agent/processmanager/processinfo.go, Line: 290, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/processmanager.(*ProcessManager).processNewExecMapping, File: /agent/processmanager/processinfo.go, Line: 427, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/processmanager.(*ProcessManager).synchronizeMappings, File: /agent/processmanager/processinfo.go, Line: 535, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/processmanager.(*ProcessManager).SynchronizeProcess, File: /agent/processmanager/processinfo.go, Line: 672, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/tracer.(*Tracer).processPIDEvents, File: /agent/tracer/events.go, Line: 50, Column: 0
Instrumentation: go, Function: go.opentelemetry.io/ebpf-profiler/tracer.(*Tracer).StartPIDEventProcessor.gowrap1, File: /agent/tracer/events.go, Line: 41, Column: 0
Instrumentation: go, Function: runtime.goexit, File: /agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.6.linux-amd64/src/runtime/asm_amd64.s, Line: 1701, Column: 0
```
