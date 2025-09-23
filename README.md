# (WIP) OpenTelemetry Lazy Backend

The OTel Lazy Backend is not supposed to replace OTel Collector and its components, but is useful to
debug raw OTel signals such as profiles. The main reason it exists is that production-grade backends
that support the OTel profiles signal are not created equal and many of them loose important details.
Also, the OTel Debug Exporter is pretty useless when it dumps symbol tables separated from stack traces.

``` mermaid
flowchart LR
  opentelemetry-ebpf-profiler
  opentelemetry-lazybackend
  clickhouse@{ shape: cyl, label: "ClickHouse" }
  sqlite@{ shape: cyl, label: "SQLite" }
  stdout["Stdout"]

  opentelemetry-ebpf-profiler e1@--> opentelemetry-lazybackend
  opentelemetry-lazybackend e2@--> clickhouse
  opentelemetry-lazybackend e3@--> sqlite
  opentelemetry-lazybackend e4@--> stdout

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
git@github.com:danielpacak/opentelemetry-lazybackend.git
cd opentelemetry-lazybackend
```

```
go build
```

``` console
$ ./opentelemetry-lazybackend
--------------- New Resource Profile --------------
  container.id: 6d7d5c336f79fbfae3e76c4a25b2c3805ffa74ab4c6145d4219ec2b92185fb6f
------------------- New Profile -------------------
  ProfileID: 00000000000000000000000000000000
  Time: 2025-09-23 04:55:35.51787723 +0000 UTC
  Duration: 1970-01-01 00:00:02.840298447 +0000 UTC
  PeriodType: [cpu, nanoseconds, Unspecified]
  Period: 50000000
  Dropped attributes count: 0
  SampleType: samples
------------------- New Sample --------------------
  Timestamp[0]: 1758603335517877230 (2025-09-23 06:55:35.51787723 +0200 CEST)
  thread.name: etcd
  process.executable.name: etcd
  process.executable.path: /usr/local/bin/etcd
  process.pid: 3086
  thread.id: 11322
---------------------------------------------------
Instrumentation: kernel, Function: do_syscall_64, File: , Line: 0, Column: 0
Instrumentation: kernel, Function: entry_SYSCALL_64_after_hwframe, File: , Line: 0, Column: 0
Instrumentation: go, Function: runtime.futex, File: runtime/sys_linux_amd64.s, Line: 558, Column: 0
Instrumentation: go, Function: runtime.futexsleep, File: runtime/os_linux.go, Line: 69, Column: 0
Instrumentation: go, Function: runtime.notesleep, File: runtime/lock_futex.go, Line: 171, Column: 0
Instrumentation: go, Function: runtime.stopm, File: runtime/proc.go, Line: 1762, Column: 0
Instrumentation: go, Function: runtime.findRunnable, File: runtime/proc.go, Line: 3147, Column: 0
Instrumentation: go, Function: runtime.schedule, File: runtime/proc.go, Line: 3868, Column: 0
Instrumentation: go, Function: runtime.park_m, File: runtime/proc.go, Line: 4037, Column: 0
Instrumentation: go, Function: runtime.mcall, File: runtime/asm_amd64.s, Line: 459, Column: 0
------------------- End Sample --------------------
------------------- New Sample --------------------
  Timestamp[0]: 1758603337408146688 (2025-09-23 06:55:37.408146688 +0200 CEST)
  Timestamp[1]: 1758603337358132101 (2025-09-23 06:55:37.358132101 +0200 CEST)
  Timestamp[2]: 1758603338358175677 (2025-09-23 06:55:38.358175677 +0200 CEST)
  thread.name: etcd
  process.executable.name: etcd
  process.executable.path: /usr/local/bin/etcd
  process.pid: 3086
  thread.id: 3421
---------------------------------------------------
Instrumentation: go, Function: runtime.pidleget, File: runtime/proc.go, Line: 6569, Column: 0
Instrumentation: go, Function: runtime.findRunnable, File: runtime/proc.go, Line: 3482, Column: 0
Instrumentation: go, Function: runtime.schedule, File: runtime/proc.go, Line: 3868, Column: 0
Instrumentation: go, Function: runtime.park_m, File: runtime/proc.go, Line: 4037, Column: 0
Instrumentation: go, Function: runtime.mcall, File: runtime/asm_amd64.s, Line: 459, Column: 0
------------------- End Sample --------------------
------------------- End Profile -------------------
-------------- End Resource Profile ---------------
```
