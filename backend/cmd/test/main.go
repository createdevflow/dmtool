package main

import (
	"fmt"
	"time"
	"backend/internal/db"
	"backend/internal/repository"
)

func main() {
	database := db.Init("dmtool.db", false)
	taskRepo := repository.NewTaskRepository(database)
	tasks, err := taskRepo.FindDueScheduledTasks(time.Now().UTC())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Found tasks:", len(tasks))
	for _, t := range tasks {
		fmt.Printf("ID: %d, DueDate: %s\n", t.ID, t.DueDate.String())
	}
}
