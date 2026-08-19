
package queue

import (
    "log"
    "github.com/jmoiron/sqlx"
)

type WriteJob struct {
    SQL  string
    Args []interface{}
}

var WriteQueue = make(chan WriteJob, 10000)

func StartWriter(db *sqlx.DB) {
    go func() {
        for job := range WriteQueue {
            _, err := db.Exec(job.SQL, job.Args...)
            if err != nil {
                log.Printf("Write error: %v", err)
            }
        }
    }()
    log.Println("Async writer started")
}

func Enqueue(sql string, args ...interface{}) {
    WriteQueue <- WriteJob{SQL: sql, Args: args}
}