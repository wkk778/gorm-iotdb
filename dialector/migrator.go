package dialector

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/yourname/gorm-iotdb/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Migrator implements the GORM migrator contract for IoTDB.
type Migrator struct {
	db        *gorm.DB
	dialector Dialector
}

// AutoMigrate creates missing tables and columns for the provided models.
func (m Migrator) AutoMigrate(dst ...interface{}) error {
	for _, value := range dst {
		if !m.HasTable(value) {
			if err := m.CreateTable(value); err != nil {
				return err
			}
			continue
		}

		stmt := &gorm.Statement{DB: m.db}
		if err := stmt.Parse(value); err != nil {
			return err
		}
		for _, field := range stmt.Schema.Fields {
			if !m.HasColumn(value, field.DBName) {
				if err := m.AddColumn(value, field.DBName); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// CurrentDatabase returns the current database if the backend exposes it.
func (m Migrator) CurrentDatabase() string {
	var name string
	_ = m.db.Raw("SHOW CURRENT_DATABASE").Scan(&name).Error
	return name
}

// FullDataTypeOf returns the complete IoTDB data type clause for a field.
func (m Migrator) FullDataTypeOf(field *schema.Field) clause.Expr {
	return clause.Expr{SQL: m.dialector.DataTypeOf(field)}
}

// GetTypeAliases returns IoTDB type aliases known to the migrator.
func (m Migrator) GetTypeAliases(databaseTypeName string) []string {
	upper := strings.ToUpper(databaseTypeName)
	switch upper {
	case "TEXT", "STRING":
		return []string{"TEXT", "STRING"}
	case "INT32", "INTEGER":
		return []string{"INT32", "INTEGER"}
	case "INT64", "LONG":
		return []string{"INT64", "LONG"}
	default:
		return []string{upper}
	}
}

// CreateTable creates IoTDB tables for the provided models.
func (m Migrator) CreateTable(dst ...interface{}) error {
	for _, value := range dst {
		stmt := &gorm.Statement{DB: m.db}
		if err := stmt.Parse(value); err != nil {
			return err
		}

		definition := make([]string, 0, len(stmt.Schema.Fields))
		for _, column := range model.ParseColumns(stmt.Schema) {
			role := "FIELD"
			switch column.Role {
			case model.ColumnRoleTag:
				role = "TAG"
			case model.ColumnRoleTime:
				role = "TIME"
			}
			definition = append(definition, fmt.Sprintf("%s %s %s", stmt.Quote(column.Field.DBName), m.dialector.DataTypeOf(column.Field), role))
		}

		sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", stmt.Quote(stmt.Table), strings.Join(definition, ", "))
		if err := m.db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

// DropTable drops IoTDB tables for the provided models.
func (m Migrator) DropTable(dst ...interface{}) error {
	for _, value := range dst {
		stmt := &gorm.Statement{DB: m.db}
		if err := stmt.Parse(value); err != nil {
			return err
		}
		if err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", stmt.Quote(stmt.Table))).Error; err != nil {
			return err
		}
	}
	return nil
}

// HasTable reports whether the destination table exists.
func (m Migrator) HasTable(dst interface{}) bool {
	stmt := &gorm.Statement{DB: m.db}
	if err := stmt.Parse(dst); err != nil {
		return false
	}
	tables, err := m.GetTables()
	if err != nil {
		return false
	}
	for _, table := range tables {
		if strings.EqualFold(table, stmt.Table) {
			return true
		}
	}
	return false
}

// RenameTable renames an IoTDB table.
func (m Migrator) RenameTable(oldName, newName interface{}) error {
	return m.db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", m.quoteIdentifier(oldName), m.quoteIdentifier(newName))).Error
}

// GetTables returns all visible tables.
func (m Migrator) GetTables() ([]string, error) {
	rows, err := m.db.Raw("SHOW TABLES").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if scanErr := rows.Scan(&table); scanErr == nil {
			tables = append(tables, table)
		}
	}
	return tables, rows.Err()
}

// TableType returns the table type metadata for a model.
func (m Migrator) TableType(dst interface{}) (gorm.TableType, error) {
	if !m.HasTable(dst) {
		return nil, sql.ErrNoRows
	}
	return tableType{name: m.quoteIdentifier(dst), kind: "TABLE"}, nil
}

// AddColumn adds a new column to an existing table.
func (m Migrator) AddColumn(dst interface{}, field string) error {
	stmt := &gorm.Statement{DB: m.db}
	if err := stmt.Parse(dst); err != nil {
		return err
	}
	f, err := model.FindField(stmt.Schema, field)
	if err != nil {
		return err
	}
	return m.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", stmt.Quote(stmt.Table), stmt.Quote(f.DBName), m.dialector.DataTypeOf(f))).Error
}

// DropColumn drops a column from an existing table.
func (m Migrator) DropColumn(dst interface{}, field string) error {
	stmt := &gorm.Statement{DB: m.db}
	if err := stmt.Parse(dst); err != nil {
		return err
	}
	f, err := model.FindField(stmt.Schema, field)
	if err != nil {
		return err
	}
	return m.db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", stmt.Quote(stmt.Table), stmt.Quote(f.DBName))).Error
}

// AlterColumn alters an existing column definition.
func (m Migrator) AlterColumn(dst interface{}, field string) error {
	stmt := &gorm.Statement{DB: m.db}
	if err := stmt.Parse(dst); err != nil {
		return err
	}
	f, err := model.FindField(stmt.Schema, field)
	if err != nil {
		return err
	}
	return m.db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", stmt.Quote(stmt.Table), stmt.Quote(f.DBName), m.dialector.DataTypeOf(f))).Error
}

// MigrateColumn updates an existing column when its definition diverges.
func (m Migrator) MigrateColumn(dst interface{}, field *schema.Field, _ gorm.ColumnType) error {
	return m.AlterColumn(dst, field.DBName)
}

// MigrateColumnUnique is a no-op because IoTDB does not expose relational unique constraints.
func (m Migrator) MigrateColumnUnique(dst interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	_ = dst
	_ = field
	_ = columnType
	return nil
}

// HasColumn reports whether a column exists.
func (m Migrator) HasColumn(dst interface{}, field string) bool {
	types, err := m.ColumnTypes(dst)
	if err != nil {
		return false
	}
	for _, column := range types {
		if strings.EqualFold(column.Name(), field) {
			return true
		}
	}
	return false
}

// RenameColumn renames a column.
func (m Migrator) RenameColumn(dst interface{}, oldName, field string) error {
	stmt := &gorm.Statement{DB: m.db}
	if err := stmt.Parse(dst); err != nil {
		return err
	}
	return m.db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", stmt.Quote(stmt.Table), stmt.Quote(oldName), stmt.Quote(field))).Error
}

// ColumnTypes returns best-effort column metadata from DESCRIBE.
func (m Migrator) ColumnTypes(dst interface{}) ([]gorm.ColumnType, error) {
	stmt := &gorm.Statement{DB: m.db}
	if err := stmt.Parse(dst); err != nil {
		return nil, err
	}

	rows, err := m.db.Raw(fmt.Sprintf("DESCRIBE %s", stmt.Quote(stmt.Table))).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []gorm.ColumnType
	for rows.Next() {
		var name, kind string
		if scanErr := rows.Scan(&name, &kind); scanErr != nil {
			return nil, scanErr
		}
		columns = append(columns, columnType{name: name, databaseType: kind})
	}
	return columns, rows.Err()
}

// CreateView returns an unsupported operation error.
func (m Migrator) CreateView(name string, option gorm.ViewOption) error {
	_ = name
	_ = option
	return errors.New("iotdb migrator: views are not supported")
}

// DropView returns an unsupported operation error.
func (m Migrator) DropView(name string) error {
	_ = name
	return errors.New("iotdb migrator: views are not supported")
}

// CreateConstraint is a no-op for IoTDB.
func (m Migrator) CreateConstraint(dst interface{}, name string) error {
	_ = dst
	_ = name
	return nil
}

// DropConstraint is a no-op for IoTDB.
func (m Migrator) DropConstraint(dst interface{}, name string) error {
	_ = dst
	_ = name
	return nil
}

// HasConstraint reports false because IoTDB does not expose relational constraints.
func (m Migrator) HasConstraint(dst interface{}, name string) bool {
	_ = dst
	_ = name
	return false
}

// CreateIndex is a no-op for IoTDB.
func (m Migrator) CreateIndex(dst interface{}, name string) error {
	_ = dst
	_ = name
	return nil
}

// DropIndex is a no-op for IoTDB.
func (m Migrator) DropIndex(dst interface{}, name string) error {
	_ = dst
	_ = name
	return nil
}

// HasIndex reports false because IoTDB does not expose relational indexes.
func (m Migrator) HasIndex(dst interface{}, name string) bool {
	_ = dst
	_ = name
	return false
}

// RenameIndex is a no-op for IoTDB.
func (m Migrator) RenameIndex(dst interface{}, oldName, newName string) error {
	_ = dst
	_ = oldName
	_ = newName
	return nil
}

// GetIndexes returns an empty set because IoTDB does not expose relational indexes.
func (m Migrator) GetIndexes(dst interface{}) ([]gorm.Index, error) {
	_ = dst
	return []gorm.Index{}, nil
}

func (m Migrator) quoteIdentifier(value interface{}) string {
	switch v := value.(type) {
	case string:
		return `"` + strings.Trim(v, `"`) + `"`
	default:
		stmt := &gorm.Statement{DB: m.db}
		if err := stmt.Parse(v); err != nil {
			return `""`
		}
		return stmt.Quote(stmt.Table)
	}
}

type tableType struct {
	name string
	kind string
}

func (t tableType) Schema() string          { return "" }
func (t tableType) Name() string            { return t.name }
func (t tableType) Type() string            { return t.kind }
func (t tableType) Comment() (string, bool) { return "", false }

type columnType struct {
	name         string
	databaseType string
}

func (c columnType) Name() string                      { return c.name }
func (c columnType) DatabaseTypeName() string          { return c.databaseType }
func (c columnType) ColumnType() (string, bool)        { return c.databaseType, true }
func (c columnType) PrimaryKey() (bool, bool)          { return false, false }
func (c columnType) AutoIncrement() (bool, bool)       { return false, false }
func (c columnType) Length() (int64, bool)             { return 0, false }
func (c columnType) DecimalSize() (int64, int64, bool) { return 0, 0, false }
func (c columnType) Nullable() (bool, bool)            { return true, true }
func (c columnType) Unique() (bool, bool)              { return false, false }
func (c columnType) ScanType() reflect.Type            { return reflect.TypeOf("") }
func (c columnType) Comment() (string, bool)           { return "", false }
func (c columnType) DefaultValue() (string, bool)      { return "", false }
