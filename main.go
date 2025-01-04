package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdheshnayak/govelte/pkg/element"
	"github.com/abdheshnayak/govelte/pkg/gocode"
	"github.com/abdheshnayak/govelte/pkg/utils"
)

func main() {
	src := flag.String("src", "", "source directory")
	dist := flag.String("dist", "", "destination directory")
	flag.Parse()

	if src == nil || dist == nil {
		flag.PrintDefaults()
		return
	}

	if *src == "" || *dist == "" {
		flag.PrintDefaults()
		return
	}

	if err := Run(*src, *dist); err != nil {
		fmt.Println("Error:", err)
	}
}

func Run(src, dist string) error {
	fmt.Println("transpiling...")
	defer fmt.Println("transpiled")

	input, err := utils.WalkAndConcatenateHTML(src)
	if err != nil {
		return err
	}

	// Parse the template
	tmpl, err := template.New("sections").Parse(string(input))
	if err != nil {
		return err
	}

	// Map to store section names and their content
	sections := make(map[string]string)

	// Execute the template for each section and store its content
	sectionNames := []string{}
	t := tmpl.Templates()
	for _, v := range t {
		sectionNames = append(sectionNames, v.Name())
	}

	for _, section := range sectionNames {
		// string writer
		var buffer strings.Builder

		err := tmpl.ExecuteTemplate(&buffer, section, nil)
		if err != nil {
			return err
		}
		sections[section] = buffer.String()
	}

	goCodes := map[string][]byte{}

	// Output the parsed map
	for name, content := range sections {
		h, err := element.NewHtml([]byte(content))
		if err != nil {
			return err
		}
		out, err := h.RenderGolangCode()
		if err != nil {
			return err
		}

		goCodes[name] = out
	}

	s, err := utils.GetPaths(src, ".go")
	if err != nil {
		return err
	}

	for _, v := range s {
		b, err := os.ReadFile(v)
		if err != nil {
			return err
		}
		out, err := gocode.ReplaceRenderE(b, goCodes)
		if err != nil {
			return err
		}

		out = append([]byte("// generated by govelte\n\n"), out...)

		outPath := strings.Replace(v, src, dist, 1)
		os.MkdirAll(filepath.Dir(outPath), 0755)

		if err := os.WriteFile(outPath, out, 0644); err != nil {
			return err
		}
	}

	return nil
}
