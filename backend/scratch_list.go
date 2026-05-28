package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Project struct {
	ID       uint
	IGHandle string
	URL      string
}

func main() {
	db, err := gorm.Open(sqlite.Open("dmtool.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var projects []Project
	db.Find(&projects)

	fmt.Println("Projects:")
	for _, p := range projects {
		fmt.Printf("ID: %d, Handle: %s, URL: %s\n", p.ID, p.IGHandle, p.URL)
	}
}
