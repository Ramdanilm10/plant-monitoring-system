package config

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB


func ConnectDatabase(){

	host := os.Getenv("SUPABASE_DB_HOST")
	port := os.Getenv("SUPABASE_DB_PORT")
	user := os.Getenv("SUPABASE_DB_USER")
	password := os.Getenv("SUPABASE_DB_PASSWORD")
	dbname := os.Getenv("SUPABASE_DB_NAME")


	connection := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host,
		port,
		user,
		password,
		dbname,
	)


	db, err := sql.Open(
		"postgres",
		connection,
	)


	if err != nil {
		panic(err)
	}


	err = db.Ping()


	if err != nil {
		panic(err)
	}


	DB = db

	fmt.Println("Database Connected")

}