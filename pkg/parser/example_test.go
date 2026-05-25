package parser_test

import (
	"fmt"

	"github.com/donaldgifford/mdp/pkg/parser"
)

func ExampleNew() {
	p := parser.New()
	html, err := p.Render([]byte("# Hi"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(string(html))
	// Output: <h1 id="hi" data-source-line="1">Hi</h1>
}
