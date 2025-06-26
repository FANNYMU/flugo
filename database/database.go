package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"flugo.com/config"
	"flugo.com/logger"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn   *sql.DB
	config *config.DatabaseConfig
}

type QueryBuilder struct {
	db          *DB
	table       string
	selectCols  []string
	whereConds  []string
	whereArgs   []interface{}
	orderBy     string
	limitCount  int
	offsetCount int
	joins       []string
}

var DefaultDB *DB

func Init(cfg *config.DatabaseConfig) {
	var err error
	DefaultDB, err = NewDB(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}
}

func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	var dsn string

	switch cfg.Driver {
	case "sqlite3", "sqlite":
		if cfg.Database == "" {
			cfg.Database = "storage/database.db"
		}
		dsn = cfg.Database
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode)
	default:
		dsn = cfg.Database
	}

	conn, err := sql.Open(cfg.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxIdleConns(cfg.MaxIdle)
	conn.SetMaxOpenConns(cfg.MaxOpen)
	conn.SetConnMaxLifetime(time.Hour)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn, config: cfg}

	if cfg.Driver == "sqlite3" || cfg.Driver == "sqlite" {
		db.createDefaultTables()
	}

	logger.Info("Database connected successfully: %s", cfg.Driver)
	return db, nil
}

func (db *DB) createDefaultTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			phone VARCHAR(20),
			age INTEGER,
			website VARCHAR(255),
			password VARCHAR(255) NOT NULL,
			avatar VARCHAR(255),
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			slug VARCHAR(255) UNIQUE,
			status VARCHAR(20) DEFAULT 'draft',
			published_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS post_categories (
			post_id INTEGER,
			category_id INTEGER,
			PRIMARY KEY (post_id, category_id),
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			logger.Error("Failed to create table: %v", err)
		}
	}

	db.seedDefaultData()
}

func (db *DB) seedDefaultData() {
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)

	if count == 0 {
		users := []string{
			"INSERT INTO users (name, email, password, age, website) VALUES ('John Doe', 'john@example.com', 'password123', 30, 'https://john.dev')",
			"INSERT INTO users (name, email, password, age) VALUES ('Jane Smith', 'jane@example.com', 'password123', 25)",
			"INSERT INTO users (name, email, password, age) VALUES ('Bob Wilson', 'bob@example.com', 'password123', 35)",
		}

		for _, query := range users {
			db.conn.Exec(query)
		}

		categories := []string{
			"INSERT INTO categories (name, description) VALUES ('Technology', 'Tech related posts')",
			"INSERT INTO categories (name, description) VALUES ('Lifestyle', 'Life and style posts')",
			"INSERT INTO categories (name, description) VALUES ('Business', 'Business and finance posts')",
		}

		for _, query := range categories {
			db.conn.Exec(query)
		}

		logger.Info("Default data seeded successfully")
	}
}

func (db *DB) Query() *QueryBuilder {
	return &QueryBuilder{
		db:         db,
		selectCols: []string{"*"},
		whereConds: []string{},
		whereArgs:  []interface{}{},
		joins:      []string{},
	}
}

func (qb *QueryBuilder) Table(table string) *QueryBuilder {
	qb.table = table
	return qb
}

func (qb *QueryBuilder) Select(cols ...string) *QueryBuilder {
	qb.selectCols = cols
	return qb
}

func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.whereConds = append(qb.whereConds, condition)
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

func (qb *QueryBuilder) Join(join string) *QueryBuilder {
	qb.joins = append(qb.joins, join)
	return qb
}

func (qb *QueryBuilder) OrderBy(order string) *QueryBuilder {
	qb.orderBy = order
	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limitCount = limit
	return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offsetCount = offset
	return qb
}

func (qb *QueryBuilder) Get() (*sql.Rows, error) {
	query := qb.buildSelectQuery()
	return qb.db.conn.Query(query, qb.whereArgs...)
}

func (qb *QueryBuilder) First() *sql.Row {
	qb.limitCount = 1
	query := qb.buildSelectQuery()
	return qb.db.conn.QueryRow(query, qb.whereArgs...)
}

func (qb *QueryBuilder) Count() (int, error) {
	oldCols := qb.selectCols
	qb.selectCols = []string{"COUNT(*)"}
	query := qb.buildSelectQuery()
	qb.selectCols = oldCols

	var count int
	err := qb.db.conn.QueryRow(query, qb.whereArgs...).Scan(&count)
	return count, err
}

func (qb *QueryBuilder) buildSelectQuery() string {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(qb.selectCols, ", "), qb.table)

	if len(qb.joins) > 0 {
		query += " " + strings.Join(qb.joins, " ")
	}

	if len(qb.whereConds) > 0 {
		query += " WHERE " + strings.Join(qb.whereConds, " AND ")
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limitCount > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limitCount)
	}

	if qb.offsetCount > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offsetCount)
	}

	return query
}

func (qb *QueryBuilder) Insert(data map[string]interface{}) (int64, error) {
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		cols = append(cols, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	result, err := qb.db.conn.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (qb *QueryBuilder) Update(data map[string]interface{}) (int64, error) {
	setParts := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		setParts = append(setParts, col+" = ?")
		values = append(values, val)
	}

	values = append(values, qb.whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s", qb.table, strings.Join(setParts, ", "))

	if len(qb.whereConds) > 0 {
		query += " WHERE " + strings.Join(qb.whereConds, " AND ")
	}

	result, err := qb.db.conn.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (qb *QueryBuilder) Delete() (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s", qb.table)

	if len(qb.whereConds) > 0 {
		query += " WHERE " + strings.Join(qb.whereConds, " AND ")
	}

	result, err := qb.db.conn.Exec(query, qb.whereArgs...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

func (db *DB) QueryRows(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Begin() (*sql.Tx, error) {
	return db.conn.Begin()
}

func Query() *QueryBuilder {
	return DefaultDB.Query()
}

func Exec(query string, args ...interface{}) (sql.Result, error) {
	return DefaultDB.Exec(query, args...)
}

func QueryRow(query string, args ...interface{}) *sql.Row {
	return DefaultDB.QueryRow(query, args...)
}

func QueryRows(query string, args ...interface{}) (*sql.Rows, error) {
	return DefaultDB.QueryRows(query, args...)
}

func ScanToStruct(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	sliceValue := destValue.Elem()
	elemType := sliceValue.Type().Elem()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		elemPtr := reflect.New(elemType)
		elem := elemPtr.Elem()

		values := make([]interface{}, len(columns))
		for i, col := range columns {
			field := elem.FieldByNameFunc(func(name string) bool {
				return strings.EqualFold(name, col)
			})
			if field.IsValid() {
				values[i] = field.Addr().Interface()
			} else {
				var dummy interface{}
				values[i] = &dummy
			}
		}

		if err := rows.Scan(values...); err != nil {
			return err
		}

		sliceValue.Set(reflect.Append(sliceValue, elem))
	}

	return rows.Err()
}
