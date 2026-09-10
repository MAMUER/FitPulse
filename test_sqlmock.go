package main

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
)

func main() {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var days []int = nil
	mock.ExpectQuery("SELECT").
		WithArgs(days).
		WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow(1))

	var x int
	err = db.QueryRow("SELECT $1", days).Scan(&x)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Success:", x)
}
