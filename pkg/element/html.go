package element

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/abdheshnayak/gohtmlx/pkg/utils"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type CompInfo struct {
	Name  string
	Props map[string]string
}

type Html interface {
	RenderGolangCode(comps map[string]CompInfo) (string, error)
}

type htmlc struct {
	nodes []*html.Node
}

func (h htmlc) RenderGolangCode(comps map[string]CompInfo) (string, error) {
	// string writer
	var buffer strings.Builder
	bts := []string{}

	for _, n := range h.nodes {
		b, err := render(n, comps)
		if err != nil {
			return "", err
		}

		if len(strings.TrimSpace(b)) == 0 {
			continue
		}

		bts = append(bts, b)
		// buffer.Write(b)
	}

	buffer.WriteString(strings.Join(bts, ","))

	return buffer.String(), nil
}

func NewHtml(htmlCode []byte) (Html, error) {
	context := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Div,
		Data:     "div",
	}

	if bytes.Contains(htmlCode, []byte("<html>")) && bytes.Contains(htmlCode, []byte("</html>")) {
		context = nil
	}

	n, err := html.ParseFragment(bytes.NewReader(bytes.Trim(bytes.TrimSpace(htmlCode), "\n")), context)
	if err != nil {
		return nil, err
	}

	return htmlc{
		nodes: n,
	}, nil
}

func trimString(s string) string {
	return strings.Trim(strings.TrimSpace(s), "\n")
}
func trimBytes(b []byte) []byte {
	return bytes.Trim(bytes.TrimSpace(b), "\n")
}

func processNode(input string) string {
	// Regular expression to match {item} or {{item}}
	input = strings.TrimSpace(input)
	varPattern := regexp.MustCompile(`\{{2,}(.*)\}{2,}`)
	tokens := []string{}

	// Split the input string into parts based on variable patterns
	splitParts := varPattern.Split(input, -1)
	matches := varPattern.FindAllStringSubmatch(input, -1)

	// Iterate over the split parts and matches to construct the result
	for i, part := range splitParts {
		if strings.TrimSpace(part) != "" {
			tokens = append(tokens, processNodePart(part))
		}
		if i < len(matches) {
			match := matches[i]
			tokens = append(tokens, fmt.Sprintf("`%s`", match[0]))
		}
	}

	inners := strings.Join(tokens, ", ")
	if len(strings.TrimSpace(inners)) == 0 {
		return ""
	}
	// Join tokens to form the final R(...) string
	result := fmt.Sprintf("R(%s)", strings.Join(tokens, ", "))
	return result
}

func processNodePart(input string) string {
	// Regular expression to match {item} or {{item}}
	varPattern := regexp.MustCompile(`\{(.*)\}`)
	tokens := []string{}

	// Split the input string into parts based on variable patterns
	splitParts := varPattern.Split(input, -1)
	matches := varPattern.FindAllStringSubmatch(input, -1)

	// Iterate over the split parts and matches to construct the result
	for i, part := range splitParts {
		if strings.TrimSpace(part) != "" {
			tokens = append(tokens, fmt.Sprintf("`%s`", part))
		}
		if i < len(matches) {
			match := matches[i]
			if strings.HasPrefix(match[0], "{") && strings.HasSuffix(match[0], "}") {
				tokens = append(tokens, fmt.Sprintf("%s", match[1]))
			} else {
				tokens = append(tokens, match[0])
			}
		}
	}

	inners := strings.Join(tokens, ", ")
	if len(strings.TrimSpace(inners)) == 0 {
		return ""
	}
	result := fmt.Sprintf("R(%s)", inners)
	return result
}

func render(n *html.Node, comps map[string]CompInfo) (string, error) {
	var buffer strings.Builder

	switch n.Type {
	case html.TextNode:
		buffer.WriteString(processNode(strings.TrimSpace(n.Data)))

	// case html.CommentNode:
	// 	buffer.WriteString("<!--")
	// 	buffer.WriteString(n.Data)
	// 	buffer.WriteString("-->")
	// case html.DoctypeNode:
	// 	buffer.WriteString("<!DOCTYPE ")
	// 	buffer.WriteString(n.Data)
	// 	buffer.WriteString(">")
	case html.ElementNode:
		isStd := isStandard(strings.TrimSpace(n.Data))
		re := regexp.MustCompile(`\{(.*)\}`)

		if isStd {
			buffer.WriteString("E(`")
			buffer.WriteString(strings.TrimSpace(n.Data))
			buffer.WriteString("`,")
			buffer.WriteString("Attrs{")
		} else {
			buffer.WriteString(fmt.Sprintf("%s(", comps[trimString(n.Data)].Name))
			buffer.WriteString(fmt.Sprintf("%sProps{", comps[trimString(n.Data)].Name))
		}

		for _, a := range n.Attr {
			if isStd {
				buffer.WriteString(fmt.Sprintf("`%s`:", a.Key))
			} else {
				if attr, ok := comps[n.Data].Props[a.Key]; ok {
					buffer.WriteString(fmt.Sprintf("%s:", utils.Capitalize(attr)))
				} else {
					buffer.WriteString(fmt.Sprintf("%s:", utils.Capitalize(a.Key)))
				}
			}
			// like {data} then remove {}
			if re.MatchString(a.Val) {
				buffer.WriteString(re.FindStringSubmatch(a.Val)[1])
			} else {
				buffer.WriteString(fmt.Sprintf("`%s`", a.Val))
			}
			buffer.WriteString(",")
		}
		buffer.WriteString("},")
		childs := []string{}
		if n.Data == "script" {
			// childs = append(childs, "E(``)")

			buffer.WriteString("R(`")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				buffer.WriteString(c.Data)
			}

			buffer.WriteString("`))")
		} else {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				b, err := render(c, comps)
				if err != nil {
					return "", err
				}

				if len(strings.TrimSpace(b)) == 0 {
					continue
				}

				childs = append(childs, string(b))
			}

			buffer.WriteString(strings.Join(childs, ","))
			buffer.WriteString(")")
		}

	case html.DocumentNode:

		childs := []string{}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			b, err := render(c, comps)
			if err != nil {
				return "", err
			}

			if len(strings.TrimSpace(b)) == 0 {
				continue
			}

			childs = append(childs, string(b))
		}

		buffer.WriteString(strings.Join(childs, ","))
	default:
		return "", nil
	}

	return buffer.String(), nil
}
