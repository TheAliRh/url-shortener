package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// quoteIdent safely quotes a table or column name. Identifiers cannot be passed as $n parameters.
func quoteIdent(name string) string {
	return pgx.Identifier{name}.Sanitize()
}

const pgUniqueViolation = "23505"

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

func addOneToDatabase(document any, tableName string) error { // Add one document to the database

	value := reflect.ValueOf(document)
	typ := reflect.TypeOf(document)

	if value.Kind() == reflect.Ptr {
		value = value.Elem()
		typ = typ.Elem()
	}

	if value.Kind() != reflect.Struct {
		log.Print("[ERROR] document must be a struct or a pointer to a struct")
		return errors.New("Could not add document to database")
	}

	var columns, placeholders []string
	var values []any

	placeholderIndex := 1
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := typ.Field(i)

		columnName := fieldType.Tag.Get("db")
		if columnName == "" {
			columnName = strings.ToLower(fieldType.Name)
		}

		if columnName == "id" && field.Int() == 0 {
			continue
		}

		columns = append(columns, quoteIdent(columnName))
		placeholders = append(placeholders, "$"+strconv.Itoa(placeholderIndex))
		values = append(values, field.Interface())
		placeholderIndex++
	}

	query := "INSERT INTO " + quoteIdent(tableName) + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

	_, err := dbPool.Exec(context.Background(), query, values...)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("insert into %s: %w: %w", tableName, ErrDuplicate, err)
		}
		return fmt.Errorf("insert into %s: %w", tableName, err)
	}

	log.Printf("Successfully added document to %s table\n", tableName)

	return nil

}

func getOneFromDatabase(tableName, condition string, document any, args ...any) (*URL, error) {

	doc, ok := document.(*URL)
	if !ok || doc == nil {
		return nil, fmt.Errorf("getOneFromDatabase: document must be a non-nil *URL, got %T", document)
	}

	query := "SELECT * FROM " + quoteIdent(tableName) + " WHERE " + condition + " LIMIT 1"

	row := dbPool.QueryRow(context.Background(), query, args...)

	value := reflect.ValueOf(doc).Elem()
	scanValues := make([]any, value.NumField())
	for i := range scanValues {
		scanValues[i] = value.Field(i).Addr().Interface()
	}

	if err := row.Scan(scanValues...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan %s row: %w", tableName, err)
	}

	log.Printf("Successfully retrieved document from %s table\n", tableName)

	return doc, nil
}

func getManyFromDatabase(tableName, condition string, args ...any) (pgx.Rows, error) {
	query := "SELECT * FROM " + quoteIdent(tableName) + " WHERE " + condition
	return dbPool.Query(context.Background(), query, args...)
}

func getAllFromDatabase(tableName string) (pgx.Rows, error) { // Get all of the database table's documents

	query := "SELECT * FROM " + quoteIdent(tableName)

	docs, err := dbPool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return docs, nil
}

func deleteOneFromDatabase(condition string, tableName string, args ...any) (string, error) {

	query := "DELETE FROM " + quoteIdent(tableName) + " WHERE " + condition

	tag, err := dbPool.Exec(context.Background(), query, args...)
	if err != nil {
		return "", fmt.Errorf("delete from %s: %w", tableName, err)
	}

	if tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}

	return "1", nil
}

func deleteManyFromDatabase(qry, tableName string) (string, error) { // Delete many documents

	query := "DELETE FROM " + quoteIdent(tableName) + " WHERE " + qry

	tag, err := dbPool.Exec(context.Background(), query)
	if err != nil {
		return "", fmt.Errorf("delete from %s: %w", tableName, err)
	}
	if tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}

	return fmt.Sprintf("%d", tag.RowsAffected()), nil
}
