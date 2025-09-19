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
2025/09/19 12:13:18 INFO starting profiles lazy backend server pid=22712 uid=1000 gid=1000
--------------- New Resource Profile --------------
  container.id: ca893606007eabf5d019726e7cfc9a1b888615a4e1a9fbb6090794462009ae83
------------------- New Profile -------------------
  ProfileID: 00000000000000000000000000000000
  Dropped attributes count: 0
  SampleType: samples
------------------- New Sample --------------------
  timestamp[0]: 1758276802457210575 (2025-09-19 12:13:22.457210575 +0200 CEST)
  thread.name: kube-controller
  process.executable.name: kube-controller-manager
  process.executable.path: /usr/local/bin/kube-controller-manager
  process.pid: 2594
  thread.id: 6623
---------------------------------------------------
Instrumentation: kernel, Function: __rcu_read_lock, File: , Line: 0, Column: 0
Instrumentation: kernel, Function: task_work_run, File: , Line: 0, Column: 0
Instrumentation: kernel, Function: syscall_exit_to_user_mode, File: , Line: 0, Column: 0
Instrumentation: kernel, Function: do_syscall_64, File: , Line: 0, Column: 0
Instrumentation: kernel, Function: entry_SYSCALL_64_after_hwframe, File: , Line: 0, Column: 0
Instrumentation: go, Function: internal/runtime/syscall.Syscall6, File: internal/runtime/syscall/asm_linux_amd64.s, Line: 36, Column: 0
Instrumentation: go, Function: internal/runtime/syscall.EpollWait, File: internal/runtime/syscall/syscall_linux.go, Line: 33, Column: 0
Instrumentation: go, Function: runtime.netpoll, File: runtime/netpoll_epoll.go, Line: 117, Column: 0
Instrumentation: go, Function: runtime.findRunnable, File: runtime/proc.go, Line: 3581, Column: 0
Instrumentation: go, Function: runtime.schedule, File: runtime/proc.go, Line: 3996, Column: 0
Instrumentation: go, Function: runtime.park_m, File: runtime/proc.go, Line: 4104, Column: 0
Instrumentation: go, Function: runtime.mcall, File: runtime/asm_amd64.s, Line: 463, Column: 0
------------------- End Sample --------------------
------------------- End Profile -------------------
-------------- End Resource Profile ---------------
```
