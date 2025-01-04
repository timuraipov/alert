package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/storage"
)

type DB struct {
	conn *sql.DB
}

func New(dsn string) *DB {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	return &DB{
		conn: db,
	}
}
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}
func (db *DB) Bootstrap(ctx context.Context) error {
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
				delta integer,
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
		fmt.Println(err)
		return metric.Metrics{}, err
	}
	err = tx.Commit()
	if err != nil {
		return metric.Metrics{}, err
	}
	return m, err
}
func (db *DB) GetAll(ctx context.Context) ([]metric.Metrics, error) {
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
	row := db.conn.QueryRowContext(ctx, `
		select name, type, delta, value from metrics where name = $1 and type = $2
	`, metricName, metricType)
	var m metric.Metrics
	err := row.Scan(&m.ID, &m.MType, &m.Delta, &m.Value)
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
