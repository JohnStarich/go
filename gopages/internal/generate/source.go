package generate

import (
	"bytes"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// customizeSourceCodePage re-renders the given source code page's HTML with fixed links and content.
// This is necessary because the source HTML is only generated in private functions inside godoc.
// * Prepends baseURL to all links.
// * Removes "View as plain text", since the generated link only adds a query param to the same page.
func customizeSourceCodePage(baseURL string, page []byte) ([]byte, error) {
	node, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, err
	}

	addBaseURL(baseURL, node)
	removeViewAsPlainText(node)

	var buf bytes.Buffer
	err = html.Render(&buf, node)
	return buf.Bytes(), err
}

func removeViewAsPlainText(node *html.Node) (deleteNode bool) {
	if node.Type == html.ElementNode && node.Data == "a" &&
		node.FirstChild != nil && node.FirstChild.Type == html.TextNode && node.FirstChild.Data == "View as plain text" {
		return true
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		shouldRemove := removeViewAsPlainText(child)
		if shouldRemove {
			node.RemoveChild(child)
		}
	}
	return false
}

func addBaseURL(baseURL string, node *html.Node) {
	if node.Type == html.ElementNode && node.Data == "a" {
		for i := range node.Attr {
			attr := &node.Attr[i]
			if attr.Key == "href" && strings.HasPrefix(attr.Val, "/") && !strings.HasPrefix(attr.Val, baseURL) {
				attr.Val = path.Join(baseURL, attr.Val)
				break
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		addBaseURL(baseURL, child)
	}
}
