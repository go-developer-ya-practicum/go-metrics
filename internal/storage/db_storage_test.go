package storage

import (
	"context"
	"sort"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/openlyinc/pointy"
	"github.com/stretchr/testify/require"

	"github.com/hikjik/go-metrics/internal/metrics"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		m        metrics.Metric
		err      error
		sqlQuery string
		target   interface{}
	}{
		{
			name:     "Get Counter",
			m:        metrics.Metric{ID: "PollCount", MType: metrics.CounterType},
			err:      nil,
			sqlQuery: "SELECT value FROM counter",
			target:   int64(1),
		},
		{
			name:     "Get Gauge",
			m:        metrics.Metric{ID: "RandomValue", MType: metrics.GaugeType},
			err:      nil,
			sqlQuery: "SELECT value FROM gauge",
			target:   1.0,
		},
		{
			name: "Unknown metric",
			m:    metrics.Metric{MType: "Unknown"},
			err:  ErrUnknownMetricType,
		},
		{
			name:     "Not found counter",
			m:        metrics.Metric{MType: metrics.GaugeType},
			err:      ErrNotFound,
			sqlQuery: "SELECT value FROM counter",
		},
		{
			name:     "Not found gauge",
			m:        metrics.Metric{MType: metrics.CounterType},
			err:      ErrNotFound,
			sqlQuery: "SELECT value FROM gauge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			storage := &DBStorage{db: db}

			mock.ExpectQuery(tt.sqlQuery).
				WithArgs(tt.m.ID).
				WillReturnRows(mock.NewRows([]string{"value"}).AddRow(tt.target))

			err = storage.Get(context.Background(), &tt.m)
			if tt.err == nil {
				require.NoError(t, err)
				require.NoError(t, mock.ExpectationsWereMet())
				switch tt.m.MType {
				case metrics.CounterType:
					require.NotNil(t, tt.m.Delta)
					require.Equal(t, *tt.m.Delta, tt.target)
				case metrics.GaugeType:
					require.NotNil(t, tt.m.Value)
					require.Equal(t, *tt.m.Value, tt.target)
				default:
					require.False(t, true)
				}
			} else {
				require.ErrorIs(t, err, tt.err)
			}
		})
	}
}

func TestList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &DBStorage{db: db}

	gauges := []*metrics.Metric{
		{ID: "RandomValue", MType: metrics.GaugeType, Value: pointy.Float64(1.0)},
		{ID: "Alloc", MType: metrics.GaugeType, Value: pointy.Float64(2.0)},
	}

	gRows := mock.NewRows([]string{"name", "value"})
	for _, g := range gauges {
		gRows = gRows.AddRow(g.ID, *g.Value)
	}
	mock.ExpectQuery("SELECT name, value FROM gauge").
		WillReturnRows(gRows)

	counters := []*metrics.Metric{
		{ID: "PollCount", MType: metrics.CounterType, Delta: pointy.Int64(1)},
		{ID: "Counter", MType: metrics.CounterType, Delta: pointy.Int64(2)},
	}
	cRows := mock.NewRows([]string{"name", "value"})
	for _, c := range counters {
		cRows = cRows.AddRow(c.ID, *c.Delta)
	}
	mock.ExpectQuery("SELECT name, value FROM counter").
		WillReturnRows(cRows)

	expected := append(gauges, counters...)

	actual, err := storage.List(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, len(expected), len(actual))

	sort.SliceStable(expected, func(i, j int) bool {
		return expected[i].ID < expected[j].ID
	})
	sort.SliceStable(actual, func(i, j int) bool {
		return actual[i].ID < actual[j].ID
	})
	for i := range actual {
		require.Equal(t, actual[i].ID, expected[i].ID)
		require.Equal(t, actual[i].MType, expected[i].MType)
		switch actual[i].MType {
		case metrics.CounterType:
			require.Equal(t, *actual[i].Delta, *expected[i].Delta)
		case metrics.GaugeType:
			require.Equal(t, *actual[i].Value, *expected[i].Value)
		default:
			require.False(t, true)
		}
	}
}

func TestPut(t *testing.T) {
	tests := []struct {
		name     string
		m        metrics.Metric
		err      error
		sqlQuery string
	}{
		{
			name:     "Put Counter",
			m:        metrics.Metric{ID: "C", MType: metrics.CounterType, Delta: pointy.Int64(1)},
			err:      nil,
			sqlQuery: "INSERT INTO counter",
		},
		{
			name:     "Put Gauge",
			m:        metrics.Metric{ID: "G", MType: metrics.GaugeType, Value: pointy.Float64(1.0)},
			err:      nil,
			sqlQuery: "INSERT INTO gauge",
		},
		{
			name: "Unknown metric",
			m:    metrics.Metric{MType: "Unknown"},
			err:  ErrUnknownMetricType,
		},
		{
			name:     "Bad Argument counter",
			m:        metrics.Metric{MType: metrics.CounterType},
			err:      ErrBadArgument,
			sqlQuery: "INSERT INTO counter",
		},
		{
			name:     "Bad Argument gauge",
			m:        metrics.Metric{MType: metrics.GaugeType},
			err:      ErrBadArgument,
			sqlQuery: "INSERT INTO gauge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			storage := &DBStorage{db: db}

			if tt.err != nil {
				require.ErrorIs(t, tt.err, storage.Put(context.Background(), &tt.m))
			} else {
				switch tt.m.MType {
				case metrics.CounterType:
					require.NotNil(t, tt.m.Delta)
					mock.ExpectExec(tt.sqlQuery).
						WithArgs(tt.m.ID, *tt.m.Delta).
						WillReturnResult(sqlmock.NewResult(1, 1))
				case metrics.GaugeType:
					require.NotNil(t, tt.m.Value)
					mock.ExpectExec(tt.sqlQuery).
						WithArgs(tt.m.ID, *tt.m.Value).
						WillReturnResult(sqlmock.NewResult(1, 1))
				default:
					require.False(t, true)
				}
				err = storage.Put(context.Background(), &tt.m)
				require.NoError(t, err)
				require.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}
