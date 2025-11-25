package clickhouse_test

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)
import "github.com/ClickHouse/clickhouse-go/v2/lib/driver"

// just play with clickhouse driver
// run the server from the cli
func TestMe(t *testing.T) {
	var db driver.Conn

	db, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	})
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping(t.Context())
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			t.Logf("worker: %d\n", worker)
			err = InsertData(t.Context(), db)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func InsertData(ctx context.Context, db driver.Conn) error {
	batch, err := db.PrepareBatch(ctx, "INSERT INTO functions_queue")
	if err != nil {
		return err
	}
	defer batch.Close()

	// 10, 100, 1000, 10000
	for i := 0; i < 100000; i++ {
		err = batch.Append(
			"b3ac1c37-0a65-4796-8029-4f4ceb94ec37",
			"01296d77-542e-4f74-985a-7e202dad6c87",
			"53ed671f-aa8c-675c-01bf-9c40744cf7e4",
			"1cf9dc83-7986-1c70-e9d6-c241c821b690",
			"9af93ec8-fbd7-6566-0157-e8047d9f3bdf",
			//String(20),
			uuid.New().String(),
			0,
			"some file path",
			"java",
			"package path",
			"d40f46e9-5a4c-9c7a-5720-494ac70202a7",
			1761455622000+i,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		x := seededRand.Intn(len(charset))
		if x < 0 {
			x = 0
		}
		b[i] = charset[x]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}
