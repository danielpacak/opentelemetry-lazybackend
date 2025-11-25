# NOTES

timestamp = 1761455622000

```
drop table functions_queue;
drop table functions;
drop view functions_queue_mv;
```

``` sql
CREATE TABLE functions_queue (
  `tenant_id` UUID,
  `project_id` UUID,
  `image_id` UUID,
  `service_id` UUID,
  `id` UUID,
  `function_name` String,
  `line_number` UInt32,
  `file_path` String,
  `language` String,
  `package_paths` String,
  `stacktrace_id` UUID,
  `timestamp` Int64 )
ENGINE = MergeTree
PARTITION BY toYYYYMM(toDateTime(timestamp / 1000))
ORDER BY (tenant_id, project_id, image_id, service_id, function_name, file_path, id)
TTL toDateTime(timestamp / 1000) + toIntervalMonth(1)
SETTINGS index_granularity = 8192;
```

``` sql
CREATE TABLE functions (
  `tenant_id` UUID,
  `project_id` UUID,
  `image_id` UUID,
  `service_id` UUID,
  `id` UUID,
  `function_name` String,
  `line_number` UInt32,
  `file_path` String,
  `language` String,
  `package_paths` String,
  `stacktrace_id` UUID,
  `created_at` DateTime,
  `updated_at` DateTime,
  `first_seen` DateTime, 
  `last_seen` DateTime 
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (tenant_id, project_id, image_id, service_id, function_name, file_path, created_at, id)
TTL created_at + toIntervalMonth(1)
SETTINGS index_granularity = 8192;
```

``` sql
CREATE MATERIALIZED VIEW functions_queue_mv TO functions (
  `tenant_id` UUID,
  `project_id` UUID,
  `image_id` UUID,
  `service_id` UUID,
  `id` UUID, 
  `function_name` String,
  `line_number` UInt32,
  `file_path` String,
  `language` String, 
  `package_paths` String,
  `stacktrace_id` UUID,
  `last_seen` DateTime, 
  `first_seen` DateTime,
  `created_at` DateTime,
  `updated_at` DateTime
)
AS SELECT
  tenant_id, project_id, image_id, service_id, id,
  function_name,
  anyLast(line_number) AS line_number,
  anyLast(file_path) AS file_path,
  anyLast(language) AS language,
  anyLast(package_paths) AS package_paths,
  anyLast(stacktrace_id) AS stacktrace_id,
  max(toDateTime(timestamp / 1000)) AS last_seen,
  min(toDateTime(timestamp / 1000)) AS first_seen,
  toDateTime(now()) AS created_at,
  toDateTime(now()) AS updated_at
FROM functions_queue
GROUP BY tenant_id, project_id, image_id, service_id, function_name, id;
```

if we insert into functions_queue the data should be inserted into function table

``` sql
insert into functions_queue
  (tenant_id, project_id, image_id, service_id, id, function_name,stacktrace_id,timestamp)
values
  ('b3ac1c37-0a65-4796-8029-4f4ceb94ec37',
  '01296d77-542e-4f74-985a-7e202dad6c87',
  '53ed671f-aa8c-675c-01bf-9c40744cf7e4',
  '1cf9dc83-7986-1c70-e9d6-c241c821b690',
  '9af93ec8-fbd7-6566-0157-e8047d9f3bdf',
  '.slowpath',
  'd40f46e9-5a4c-9c7a-5720-494ac70202a7',
  1761455622000);
```