package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pprofile"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// TODO init to exec migrations and create tables
// TODO create config from flags
type Config struct {
	DSN string
}

func DefaultConfig() Config {
	return Config{
		DSN: "postgres://lazybackend:lazybackend@localhost:5432/lazybackend?sslmode=disable",
	}
}

type Postgres struct {
	config Config

	db *sql.DB

	models *Models
}

// TODO add volume to persist data between restarts

// NewReceiver constructs a new OpenTelemetry signals receiver that persists data in PostgreSQL database.
//
//	docker run  --rm -d --name lazybackend-db \
//	  -e POSTGRES_DB=lazybackend \
//	  -e POSTGRES_USER=lazybackend \
//	  -e POSTGRES_PASSWORD=lazybackend \
//	  -p 5432:5432 \
//	  postgres
//
// The database schema is managed with migrations
//
//	go install -tags 'postgres,clickhouse' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0
//
// Create new migration
//
//	migrate create -seq -ext=.sql -dir=./receiver/postgres/migrations create_table_linux_program_exec
func NewReceiver(config Config) *Postgres {
	return &Postgres{
		config: config,
	}
}

func (s *Postgres) Init(ctx context.Context) error {
	var err error

	s.db, err = sql.Open("postgres", s.config.DSN)
	if err != nil {
		return err
	}

	// todo add flag whether you want to ping db
	err = s.db.PingContext(ctx)
	if err != nil {
		_ = s.db.Close()
		return err
	}

	// todo add flag to enable migrations
	// todo add migrations source to config
	migrationDriver, err := postgres.WithInstance(s.db, &postgres.Config{})
	migrator, err := migrate.NewWithDatabaseInstance("file://receiver/postgres/migrations", "postgres", migrationDriver)
	if err != nil {
		return err
	}
	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	s.models = NewModels(s.db)

	return nil
}

func (s *Postgres) Close(_ context.Context) error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Postgres) ReceiveProfiles(ctx context.Context, pd pprofile.Profiles) error {
	// TODO Not implemented yet
	return nil
}

// type=dynamic_exec
// argv
// binary path
// cluster, project,tenant
func LogRecordToLinuxProgram(logRecord plog.LogRecord) (*LinuxProgram, error) {
	pidValue, _ := logRecord.Attributes().Get("pid")
	ppidValue, _ := logRecord.Attributes().Get("ppid")
	containerIDValue, _ := logRecord.Attributes().Get("containerID")

	return &LinuxProgram{
		ExecutedAt:  time.Now(), // tood parse it
		PPid:        int(ppidValue.Int()),
		Pid:         int(pidValue.Int()),
		ContainerID: containerIDValue.AsString(),
	}, nil
}

// todo pass context
func (s *Postgres) ReceiveLogs(ctx context.Context, ld plog.Logs) error {
	// TODO use pg driver and batch insert to PG
	slog.Info("Save log data in PG >>>", "batch", ld.LogRecordCount())
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLogs := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLogs.ScopeLogs().Len(); j++ {
			scopeLogs := resourceLogs.ScopeLogs().At(j)
			for k := 0; k < scopeLogs.LogRecords().Len(); k++ {
				logRecord := scopeLogs.LogRecords().At(k)
				logRecordTypeValue, ok := logRecord.Attributes().Get("type")
				if !ok {
					continue
				}
				if "dynamic_exec" != logRecordTypeValue.AsString() {
					continue
				}

				linuxProgram, err := LogRecordToLinuxProgram(logRecord)
				if err != nil {
					return err
				}
				err = s.models.LinuxPrograms.Insert(linuxProgram)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

type LinuxProgram struct {
	ExecutedAt  time.Time
	Pid         int
	PPid        int
	ContainerID string
}

type Models struct {
	LinuxPrograms LinuxProgramsModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		LinuxPrograms: LinuxProgramsModel{DB: db},
	}
}

type LinuxProgramsModel struct {
	DB *sql.DB
}

func (m *LinuxProgramsModel) Insert(process *LinuxProgram) error {
	query := `
		INSERT INTO linux_programs_exec (exec_at, ppid, pid, container_id)
		VALUES ($1, $2, $3, $4);
		`
	args := []any{process.ExecutedAt, process.PPid, process.Pid, process.ContainerID}
	_, err := m.DB.Exec(query, args...)
	return err
}
