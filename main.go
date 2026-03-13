package main

import (
	"fmt"

	"github.com/Zallu35/blog-aggregator/internal/config"
)

func main() {
	newConfig, e := config.Read()
	if e != nil {
		fmt.Printf("Error reading config file: %v", e)
	}
	er := newConfig.SetUser("Sam")
	if er != nil {
		fmt.Printf("Error setting new user: %v", er)
	}
	updatedConfig, err := config.Read()
	if err != nil {
		fmt.Printf("Error reading config file after SetUser: %v", err)
	}
	fmt.Println(updatedConfig)
}
