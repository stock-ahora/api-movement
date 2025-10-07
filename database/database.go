package database

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
)

func Connect(user, pass, host, port, dbname string) (*sql.DB, error) {
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
        host, port, user, pass, dbname)

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }

    if err := db.Ping(); err != nil {
        return nil, err
    }

    return db, nil
}