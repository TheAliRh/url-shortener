package main

import (
	"context"
	"log"
	"reflect"
	"strconv"
	"strings"
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
func addManyToDatabase(shortURLs []string, newLongURLs []string) error {
	// Implementation for adding multiple short URLs and their associated long URLs to the database
	// This is a placeholder for actual database interaction code
	return nil
}

func getOneFromDatabase(shortURL string) (string, error) {
	// Implementation for retrieving the long URL from the database using the short URL
	// This is a placeholder for actual database interaction code
	return "", nil
}

func getManyFromDatabase(shortURLs []string) (map[string]string, error) {
	// Implementation for retrieving multiple long URLs from the database using a list of short URLs
	// This is a placeholder for actual database interaction code
	return map[string]string{}, nil
}

func getAllFromDatabase() ([]string, error) {
	// Implementation for retrieving all shortened URLs from the database
	// This is a placeholder for actual database interaction code
	return []string{}, nil
}

func updateOneInDatabase(shortURL, newLongURL string) error {
	// Implementation for updating the long URL associated with a short URL in the database
	// This is a placeholder for actual database interaction code
	return nil
}

func updateManyInDatabase(shortURLs []string, newLongURLs []string) error {
	// Implementation for updating multiple long URLs associated with short URLs in the database
	// This is a placeholder for actual database interaction code
	return nil
}

func deleteOneFromDatabase(shortURL string) error {
	// Implementation for deleting the short URL and its associated long URL from the database
	// This is a placeholder for actual database interaction code
	return nil
}

func deleteManyFromDatabase(shortURLs []string) error {
	// Implementation for deleting multiple short URLs and their associated long URLs from the database
	// This is a placeholder for actual database interaction code
	return nil
}
