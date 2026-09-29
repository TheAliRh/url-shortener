package main

import (
	"context"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func addOneToDatabase(document any, tableName string) error { // Add one document to the database

	value := reflect.ValueOf(document)
	typ := reflect.TypeOf(document)

	if value.Kind() == reflect.Ptr {
		value = value.Elem()
		typ = typ.Elem()
	}

	if value.Kind() != reflect.Struct {
		log.Fatal("document must be a struct or a pointer to a struct")
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

		columns = append(columns, columnName)
		placeholders = append(placeholders, "$"+strconv.Itoa(placeholderIndex))
		values = append(values, field.Interface())
		placeholderIndex++
	}

	query := "INSERT INTO " + tableName + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

	_, err := dbPool.Exec(context.Background(), query, values...)
	if err != nil {
		log.Printf("Error executing query: %v\n", err)
		return err
	}

	log.Printf("Successfully added document to %s table\n", tableName)

	return nil

}

func getOneFromDatabase(tableName string, qry string, document any) (doc *URL, err error) { // Get a document from database table

	query := "SELECT * FROM " + tableName + " WHERE " + qry + " LIMIT 1"

	row := dbPool.QueryRow(context.Background(), query)

	value := reflect.ValueOf(document)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {

		log.Fatal("Error: document must be a pointer to a struct:")
		return nil, err

	}

	value = value.Elem()

	scanValues := make([]any, value.NumField())
	for i := 0; i < value.NumField(); i++ {
		scanValues[i] = value.Field(i).Addr().Interface()
	}

	err = row.Scan(scanValues...)
	if err != nil {
		log.Printf("Error scanning row: %v\n", err)
		return nil, err
	}

	log.Printf("Successfully retrieved document from %s table\n", tableName)

	doc, ok := document.(*URL)
	if !ok {
		return nil, err
	}

	return doc, nil
}

func getManyFromDatabase(qry, tableName string) (pgx.Rows, error) {

	query := "SELECT * FROM " + tableName + " WHERE " + qry

	response, err := dbPool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func getAllFromDatabase(tableName string) (pgx.Rows, error) { // Get all of the database table's documents

	query := "SELECT * FROM " + tableName

	docs, err := dbPool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return docs, nil
}

func deleteOneFromDatabase(doc string, tableName string) (string, error) { // Delete a document from database table

	query := "DELETE FROM " + tableName + " WHERE " + doc

	_, err := dbPool.Exec(context.Background(), query)
	if err != nil {
		return "", err
	}

	return "1", nil
}

func deleteManyFromDatabase(qry, tableName string) (string, error) { // Delete many documents

	query := "DELETE FROM " + tableName + " WHERE " + qry

	_, err := dbPool.Exec(context.Background(), query)
	if err != nil {
		return "", err
	}

	return "Removed successfully", nil
}
