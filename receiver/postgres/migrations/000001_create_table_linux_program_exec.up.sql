CREATE TABLE IF NOT EXISTS linux_programs_exec (
  exec_at timestamp(0) with time zone NOT NULL,
  ppid integer NOT NULL,
  pid integer NOT NULL,
  container_id text
);
