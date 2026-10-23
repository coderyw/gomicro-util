// Package mysql
// @Author: yinwei
// @File: db
// @Version: 1.0.0
// @Date: 2025/4/14 23:06

package mysql

import (
	"context"
	"database/sql"
)

type WrappedDB struct {
	db *sql.DB
}

func (w *WrappedDB) GetDBConn() (*sql.DB, error) {
	return w.db, nil
}

func (w *WrappedDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	//start := time.Now()
	stmt, err := w.db.PrepareContext(ctx, query)
	//duration := time.Since(start)
	//log.InfoCtx(ctx, "PrepareContext 获取连接到执行完成耗时", zap.Any("Time", duration.String()))

	return stmt, err
}

func (w *WrappedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	//start := time.Now()
	rows := w.db.QueryRowContext(ctx, query, args...)
	//duration := time.Since(start)
	//log.InfoCtx(ctx, "QueryRowContext 获取连接到执行完成耗时", zap.Any("Time", duration.String()))

	return rows
}

func (w *WrappedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	//start := time.Now()
	rows, err := w.db.QueryContext(ctx, query, args...)

	//duration := time.Since(start)
	//log.InfoCtx(ctx, "QueryContext 获取连接到执行完成耗时", zap.Any("Time", duration.String()))

	return rows, err
}

func (w *WrappedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	//start := time.Now()
	res, err := w.db.ExecContext(ctx, query, args...)
	//duration := time.Since(start)
	//log.InfoCtx(ctx, "ExecContext 获取连接到执行完成耗时", zap.Any("Time", duration.String()))
	return res, err
}
