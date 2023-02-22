package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"time"
)
func task(s string) {
	fmt.Println(s)
}

func main() {
	s := gocron.NewScheduler(time.UTC)
	s.Every(5).Seconds().Do(task,"5 Sec")
	s.StartBlocking()
}
