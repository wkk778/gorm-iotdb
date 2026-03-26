// Package dialector implements a reusable GORM dialector for Apache IoTDB.
package dialector

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/wkk778/gorm-iotdb/driver"
	"github.com/wkk778/gorm-iotdb/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// TagShardFunc resolves the physical table name that should receive a row for a tag set.
type TagShardFunc func(logicalTable string, tags map[string]any) string

// Config configures the IoTDB dialector.
type Config struct {
	DSN          string
	DriverName   string
	Conn         gorm.ConnPool
	TagShardFunc TagShardFunc
}

// Dialector implements gorm.Dialector for Apache IoTDB.
type Dialector struct {
	config Config
}

// Open creates a new IoTDB dialector from a DSN.
func Open(dsn string) gorm.Dialector {
	return New(Config{DSN: dsn})
}

// New creates a new IoTDB dialector from a structured configuration.
func New(config Config) gorm.Dialector {
	return Dialector{config: config}
}

// Name returns the GORM dialector name.
func (d Dialector) Name() string {
	return "iotdb"
}

// Initialize attaches the IoTDB connection pool and registers default callbacks.
func (d Dialector) Initialize(db *gorm.DB) error {
	if d.config.Conn != nil {
		db.ConnPool = d.config.Conn
	} else {
		if d.config.DSN == "" {
			return fmt.Errorf("iotdb gorm dialector: empty DSN")
		}

		sqlDB, err := driver.Open(driver.Config{
			DSN:        d.config.DSN,
			DriverName: d.config.DriverName,
		})
		if err != nil {
			return err
		}
		db.ConnPool = sqlDB
	}

	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	if err := db.Callback().Create().Replace("gorm:create", d.createCallback()); err != nil {
		return err
	}

	db.ClauseBuilders["LIMIT"] = buildLimit
	return nil
}

// Migrator returns the IoTDB-aware migrator.
func (d Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{db: db, dialector: d}
}

// DataTypeOf returns the IoTDB type for a GORM field.
func (d Dialector) DataTypeOf(field *schema.Field) string {
	if explicitType := strings.TrimSpace(field.TagSettings["TYPE"]); explicitType != "" {
		return explicitType
	}

	switch field.DataType {
	case schema.Bool:
		return "BOOLEAN"
	case schema.Int, schema.Uint:
		if field.Size > 0 && field.Size <= 32 {
			return "INT32"
		}
		return "INT64"
	case schema.Float:
		if field.Precision > 0 && field.Precision <= 24 {
			return "FLOAT"
		}
		return "DOUBLE"
	case schema.String:
		return "TEXT"
	case schema.Bytes:
		return "BLOB"
	case schema.Time:
		return "TIMESTAMP"
	default:
		return "TEXT"
	}
}

// DefaultValueOf returns IoTDB's default-value expression placeholder.
func (d Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	if field.DefaultValue != "" {
		return clause.Expr{SQL: field.DefaultValue}
	}
	return clause.Expr{SQL: "DEFAULT"}
}

// BindVarTo writes a positional bind variable.
func (d Dialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ interface{}) {
	_ = writer.WriteByte('?')
}

// QuoteTo writes a quoted identifier.
func (d Dialector) QuoteTo(writer clause.Writer, str string) {
	_ = writer.WriteByte('"')
	for _, r := range str {
		if r == '"' {
			_, _ = writer.WriteString(`""`)
			continue
		}
		_, _ = writer.WriteString(string(r))
	}
	_ = writer.WriteByte('"')
}

// Explain renders SQL with bound values for logs and dry runs.
func (d Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, regexp.MustCompile(`\?`), "'", vars...)
}

// SavePoint creates a transaction savepoint.
func (d Dialector) SavePoint(tx *gorm.DB, name string) error {
	return tx.Exec("SAVEPOINT " + name).Error
}

// RollbackTo rolls back a transaction savepoint.
func (d Dialector) RollbackTo(tx *gorm.DB, name string) error {
	return tx.Exec("ROLLBACK TO SAVEPOINT " + name).Error
}

func (d Dialector) createCallback() func(*gorm.DB) {
	defaultCreate := callbacks.Create(&callbacks.Config{})

	return func(db *gorm.DB) {
		if db.Error != nil || db.Statement.Schema == nil || d.config.TagShardFunc == nil {
			defaultCreate(db)
			return
		}

		groups, err := d.groupByShard(db.Statement)
		if err != nil || len(groups) <= 1 {
			if err != nil {
				_ = db.AddError(err)
				return
			}
			defaultCreate(db)
			return
		}

		var rowsAffected int64
		for _, group := range groups {
			subtx := db.Session(&gorm.Session{NewDB: true, Initialized: true, SkipHooks: true})
			subtx.Statement.Table = group.Table
			subtx.Statement.TableExpr = nil
			subtx.Statement.Schema = db.Statement.Schema
			subtx.Statement.Dest = group.Value
			subtx.Statement.ReflectValue = indirectValue(reflect.ValueOf(group.Value))
			defaultCreate(subtx)
			if subtx.Error != nil {
				_ = db.AddError(subtx.Error)
				return
			}
			rowsAffected += subtx.RowsAffected
		}
		db.RowsAffected = rowsAffected
	}
}

type shardGroup struct {
	Table string
	Value any
}

func (d Dialector) groupByShard(stmt *gorm.Statement) ([]shardGroup, error) {
	value := indirectValue(reflect.ValueOf(stmt.Dest))
	if !value.IsValid() {
		return nil, fmt.Errorf("iotdb: invalid create destination")
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return nil, nil
	}

	groups := make(map[string]reflect.Value)
	order := make([]string, 0)
	for i := 0; i < value.Len(); i++ {
		row := indirectValue(value.Index(i))
		tags := model.TagValueMap(stmt.Schema, row)
		table := d.config.TagShardFunc(stmt.Table, tags)
		if table == "" || table == stmt.Table {
			return nil, nil
		}
		group, ok := groups[table]
		if !ok {
			group = reflect.MakeSlice(value.Type(), 0, value.Len())
			order = append(order, table)
		}
		groups[table] = reflect.Append(group, value.Index(i))
	}

	result := make([]shardGroup, 0, len(order))
	for _, table := range order {
		result = append(result, shardGroup{
			Table: table,
			Value: groups[table].Interface(),
		})
	}
	return result, nil
}

func buildLimit(c clause.Clause, builder clause.Builder) {
	limit, ok := c.Expression.(clause.Limit)
	if !ok {
		c.Build(builder)
		return
	}

	if limit.Limit != nil && *limit.Limit >= 0 {
		_, _ = builder.WriteString("LIMIT ")
		builder.AddVar(builder, *limit.Limit)
	}

	if limit.Offset > 0 {
		_, _ = builder.WriteString(" OFFSET ")
		builder.AddVar(builder, limit.Offset)
	}
}

func indirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

var _ gorm.Dialector = (*Dialector)(nil)
var _ gorm.SavePointerDialectorInterface = (*Dialector)(nil)

// DB returns the underlying *sql.DB when the configured pool exposes it.
func (d Dialector) DB(conn gorm.ConnPool) (*sql.DB, bool) {
	db, ok := conn.(*sql.DB)
	return db, ok
}
