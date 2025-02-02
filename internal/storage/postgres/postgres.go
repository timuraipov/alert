package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/storage"
	"go.uber.org/zap"
)

const retryCount = 3

var retryInterval = []int{1, 3, 5}

type DB struct {
	conn *sql.DB
	dsn  string
}

func (db *DB) getConnection() error {
	err := db.conn.Ping()
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code) {
			newConn, err := connectionWithRetry(db.dsn)
			db.conn = newConn
			return err
		} else {
			return err
		}
	}
	return nil
}
func New(dsn string) (*DB, error) {
	op := "storage.postgres.new"
	db, err := connectionWithRetry(dsn)
	if err != nil {
		logger.Log.Error("trouble with connection",
			zap.String("operation", op),
			zap.Error(err),
		)
		return &DB{}, err
	}
	return &DB{conn: db, dsn: dsn}, err
}
func connectionWithRetry(dsn string) (*sql.DB, error) {
	op := "storage.postgres.connectionWithRetry"
	var (
		db  *sql.DB
		err error
	)
	var pgErr *pgconn.PgError
	db, err = sql.Open("pgx", dsn)
	if err == nil {
		return db, err
	}
	for i := 0; i < retryCount; i++ {
		logger.Log.Error("database connect problem",
			zap.String("operation", op),
			zap.String("trying to reconnect db:", fmt.Sprintf("tries number- %d", i+1)),
			zap.Error(err),
		)
		time.Sleep(time.Duration(retryInterval[i]) * time.Second)
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			if !errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code) {
				return nil, err
			}
		} else {
			return db, nil
		}

	}
	return nil, err
}
func (db *DB) Ping(ctx context.Context) error {
	err := db.getConnection()
	if err != nil {
		return err
	}
	return db.conn.PingContext(ctx)
}
func (db *DB) Bootstrap(ctx context.Context) error {
	err := db.getConnection()
	if err != nil {
		return err
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		create table if not exists metrics
		(
				id    serial
						primary key,
				name  varchar(100) not null,
				type  varchar(50)  not null,
				delta bigint,
				value double precision,
				constraint unique_metric
        unique (name, type)
		);
	`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) Save(ctx context.Context, m metric.Metrics) (metric.Metrics, error) {
	err := db.getConnection()
	if err != nil {
		return metric.Metrics{}, err
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return metric.Metrics{}, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `select name, type, delta, value from metrics where name=$1 and type=$2`, m.ID, m.MType)
	var selectedM metric.Metrics
	err = row.Scan(&selectedM.ID, &selectedM.MType, &selectedM.Delta, &selectedM.Value)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return metric.Metrics{}, err
		}
		_, err = tx.ExecContext(ctx, `
		INSERT INTO public.metrics (name, type, delta, value)
	VALUES ($1, $2, $3, $4)
		`, m.ID, m.MType, m.Delta, m.Value)
		if err != nil {
			return metric.Metrics{}, err
		}
		err := tx.Commit()
		if err != nil {
			return metric.Metrics{}, err
		}
		return m, nil
	}
	if m.MType == metric.MetricTypeGauge {
		_, err := tx.ExecContext(ctx, `update metrics SET value = $1 where name=$2 and type=$3`, m.Value, m.ID, m.MType)
		if err != nil {
			return metric.Metrics{}, err
		}
	} else {
		_, err := tx.ExecContext(ctx, `update metrics SET delta = delta+$1 where name=$2 and type=$3`, m.Delta, m.ID, m.MType)
		if err != nil {
			return metric.Metrics{}, err
		}
	}
	row = tx.QueryRowContext(ctx, `select name,type,delta,value from metrics where name=$1 and type=$2`, m.ID, m.MType)
	if err = row.Scan(&m.ID, &m.MType, &m.Delta, &m.Value); err != nil {
		return metric.Metrics{}, err
	}
	err = tx.Commit()
	if err != nil {
		return metric.Metrics{}, err
	}
	return m, err
}
func (db *DB) GetAll(ctx context.Context) ([]metric.Metrics, error) {
	err := db.getConnection()
	if err != nil {
		return nil, err
	}
	rows, err := db.conn.QueryContext(ctx, `
		select name, type, delta, value from metrics;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var metrics []metric.Metrics
	for rows.Next() {
		var m metric.Metrics
		if err := rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return metrics, nil
}
func (db *DB) GetByTypeAndName(ctx context.Context, metricType, metricName string) (metric.Metrics, error) {
	err := db.getConnection()
	if err != nil {
		return metric.Metrics{}, err
	}
	row := db.conn.QueryRowContext(ctx, `
		select name, type, delta, value from metrics where name = $1 and type = $2
	`, metricName, metricType)
	var m metric.Metrics
	err = row.Scan(&m.ID, &m.MType, &m.Delta, &m.Value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return metric.Metrics{}, storage.ErrMetricNotFound
		}
		return metric.Metrics{}, err
	}
	return m, nil
}

func (db *DB) Flush() error {
	return nil
}
func (db *DB) SaveBatch(ctx context.Context, metrics []metric.Metrics) error {
	err := db.getConnection()
	if err != nil {
		return err
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtQuery, err := tx.PrepareContext(ctx, `
	select name,type,delta,value from metrics where name=$1 and type=$2
	`)
	if err != nil {
		return err
	}
	defer stmtQuery.Close()

	stmtInsert, err := tx.PrepareContext(ctx, `
	insert into metrics (name, type, delta,value) values($1,$2,$3,$4)
	`)
	if err != nil {
		return err
	}
	defer stmtInsert.Close()

	stmtUpdateCounter, err := tx.PrepareContext(ctx, `
	update metrics set delta=delta + $1 where name=$2 and type=$3
	`)
	if err != nil {
		return err
	}
	defer stmtUpdateCounter.Close()

	stmtUpdateGauge, err := tx.PrepareContext(ctx, `
	update metrics set value=$1 where name=$2 and type=$3
	`)
	if err != nil {
		return err
	}
	defer stmtUpdateGauge.Close()

	for _, m := range metrics {
		var localM metric.Metrics

		row := stmtQuery.QueryRowContext(ctx, m.ID, m.MType)
		err = row.Scan(&localM.ID, &localM.MType, &localM.Delta, &localM.Value) // how i can check empty result without scan?
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			_, err = stmtInsert.ExecContext(ctx, m.ID, m.MType, m.Delta, m.Value)
			if err != nil {
				return err
			}

		} else {

			if m.MType == metric.MetricTypeGauge {
				_, err = stmtUpdateGauge.ExecContext(ctx, m.Value, m.ID, m.MType)
				if err != nil {
					return err
				}

			} else {
				_, err = stmtUpdateCounter.ExecContext(ctx, m.Delta, m.ID, m.MType)
				if err != nil {
					return err
				}

			}
		}
	}

	return tx.Commit()
}
