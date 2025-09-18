# OpenTelemetry Profiles Lazy Backend

```
git@github.com:danielpacak/opentelemetry-profiles-lazybackend.git
cd opentelemetry-profiles-lazybackend
```

```
go build
```

``` console
$ ./opentelemetry-profiles-lazybackend
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
