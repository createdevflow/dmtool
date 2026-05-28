package main

import (
	"fmt"
	"github.com/gocolly/colly/v2"
)

func main() {
	c := colly.NewCollector()

	c.OnHTML("title", func(e *colly.HTMLElement) {
		fmt.Println("Page title:", e.Text)
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	err := c.Visit("https://example.com/")
    if err != nil {
        fmt.Println("Error visiting:", err)
    }
}
