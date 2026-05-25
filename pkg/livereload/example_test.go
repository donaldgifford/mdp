package livereload_test

import (
	"fmt"

	"github.com/donaldgifford/mdp/pkg/livereload"
)

func ExampleHub() {
	hub := livereload.NewHub()
	defer func() { _ = hub.Close() }()

	fmt.Println(hub.Count())
	// Output: 0
}
