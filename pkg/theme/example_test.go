package theme_test

import (
	"fmt"

	"github.com/donaldgifford/mdp/pkg/theme"
)

func ExampleResolve() {
	t, err := theme.Resolve("github-light")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("mermaid=%q auto=%v hljs=%q\n", t.MermaidTheme, t.IsAuto(), t.HljsVendorCSS)
	// Output: mermaid="base" auto=false hljs="/vendor/hljs/github.min.css"
}
