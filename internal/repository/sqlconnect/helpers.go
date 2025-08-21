package sqlconnect

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

func addSorting(r *http.Request, query string) string {
	sortParams := r.URL.Query()["sort_by"] // NB! this approach for getting a slice of strings instead of one big string
	if len(sortParams) > 0 {
		var orderClauses []string
		for _, param := range sortParams {
			parts := strings.Split(param, ":")
			if len(parts) != 2 {
				continue
			}
			field, order := parts[0], parts[1]
			if !isValidSortField(field) || !isValidSortOrder(order) {
				continue
			}
			orderClauses = append(orderClauses, field+" "+order)
		}
		if len(orderClauses) > 0 {
			query += " ORDER BY " + strings.Join(orderClauses, ", ")
		}
	}
	return query
}

func isValidSortOrder(order string) bool {
	return order == "asc" || order == "desc"
}

func isValidSortField(field string) bool {
	validFields := map[string]bool{
		"first_name": true,
		"last_name":  true,
		"email":      true,
		"class":      true,
		"subject":    true,
	}
	return validFields[field]
}

func addFilters(r *http.Request, query string, args []any) (string, []any) {
	params := map[string]string{
		"first_name": "first_name",
		"last_name":  "last_name",
		"email":      "email",
		"class":      "class",
		"subject":    "subject",
	}

	for param, dbField := range params {
		value := r.URL.Query().Get(param)
		if value != "" {
			query += " AND " + dbField + " = ?"
			args = append(args, value)
		}
	}
	return query, args
}

func generateInsertQuery(model any, intoTableName string) string {
	modelType := reflect.TypeOf(model)
	var columns, placeholders string // placeholder means a questionmark in a the query string
	for i := 0; i < modelType.NumField(); i++ {
		dbTag := modelType.Field(i).Tag.Get("db")
		fmt.Println("DB tag: ", dbTag)
		dbTag = strings.TrimSuffix(dbTag, ",omitempty")
		if dbTag != "" && dbTag != "id" {
			if columns != "" {
				columns += ", "
				placeholders += ", "
			}
			columns += dbTag
			placeholders += "?"
		}
	}
	fmt.Printf("INSERT INTO %s (%s) VALUES (%s)\n", intoTableName, columns, placeholders)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", intoTableName, columns, placeholders)
}

func getStructValues(model any) []any {
	modelValue := reflect.ValueOf(model)
	modelType := modelValue.Type()
	values := []any{}
	for i := 0; i < modelType.NumField(); i++ {
		dbTag := modelType.Field(i).Tag.Get("db")
		if dbTag != "" && dbTag != "id,omitempty" {
			values = append(values, modelValue.Field(i).Interface())
		}
	}
	fmt.Println("Values:", values)
	return values
}
