package dom

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

func GetFullHTMLAction(res *string) chromedp.Action {
	return chromedp.Evaluate(`document.documentElement.outerHTML`, res)
}

func GetTextContentAction(res *string) chromedp.Action {
	return chromedp.Evaluate(`document.body.innerText`, res)
}

func GetOuterHTMLAction(selector string, res *string) chromedp.Action {
	return chromedp.OuterHTML(selector, res, chromedp.ByQuery)
}

func GetSimplifiedDOM(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = simplifyNode(&buf, doc)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func simplifyNode(w io.Writer, n *html.Node) error {
	switch n.Type {
	case html.ErrorNode:
		return nil
	case html.DocumentNode:
		// Process children
	case html.DoctypeNode:
		if _, err := io.WriteString(w, "<!DOCTYPE "+n.Data+">"); err != nil {
			return err
		}
	case html.CommentNode:
		return nil
	case html.TextNode:
		trimmed := strings.TrimSpace(n.Data)
		if trimmed != "" {
			if _, err := io.WriteString(w, html.EscapeString(trimmed)+" "); err != nil {
				return err
			}
		}
		return nil
	case html.ElementNode:
		if n.Data == "script" || n.Data == "style" || n.Data == "noscript" || n.Data == "meta" || n.Data == "link" {
			return nil
		}

		allowedTags := map[string]bool{
			"html": true, "head": true, "body": true, "title": true,
			"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
			"p": true, "div": true, "span": true, "br": false, "hr": false,
			"ul": true, "ol": true, "li": true,
			"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "th": true, "td": true,
			"a": true, "button": true, "input": false, "textarea": true, "select": true, "option": true, "label": true,
			"form": true, "img": false, "pre": true, "code": true, "strong": true, "em": true, "b": true, "i": true,
		}
		if !allowedTags[n.Data] {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if err := simplifyNode(w, c); err != nil {
					return err
				}
			}
			return nil
		}

		if _, err := io.WriteString(w, "<"+n.Data); err != nil {
			return err
		}

		allowedAttrs := map[string]bool{
			"href": true, "src": true, "alt": true, "title": true,
			"id": true, "class": true,
			"type": true, "value": true, "placeholder": true, "name": true,
			"selected": true, "checked": true, "disabled": true, "readonly": true,
			"aria-label": true, "aria-hidden": true, "role": true,
		}

		for _, a := range n.Attr {
			if allowedAttrs[a.Key] {
				val := strings.TrimSpace(a.Val)
				if val != "" || a.Key == "value" || a.Key == "selected" || a.Key == "checked" || a.Key == "disabled" || a.Key == "readonly" {
					if _, err := io.WriteString(w, " "+a.Key+"=\""+html.EscapeString(val)+"\""); err != nil {
						return err
					}
				}
			}
		}

		if _, err := io.WriteString(w, ">"); err != nil {
			return err
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := simplifyNode(w, c); err != nil {
			return err
		}
	}

	if n.Type == html.ElementNode {
		allowedTags := map[string]bool{
			"html": true, "head": true, "body": true, "title": true,
			"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
			"p": true, "div": true, "span": true, "br": false, "hr": false,
			"ul": true, "ol": true, "li": true,
			"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "th": true, "td": true,
			"a": true, "button": true, "input": false, "textarea": true, "select": true, "option": true, "label": true,
			"form": true, "img": false, "pre": true, "code": true, "strong": true, "em": true, "b": true, "i": true,
		}
		if allowed, ok := allowedTags[n.Data]; ok && allowed {
			if _, err := io.WriteString(w, "</"+n.Data+">"); err != nil {
				return err
			}
		}
	}

	return nil
}

func TypeAction(selector string, text string) chromedp.Action {
	return chromedp.SendKeys(selector, text, chromedp.ByQuery)
}

func ClickAction(selector string) chromedp.Action {
	return chromedp.Tasks{
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	}
}

func NavigateAction(url string) chromedp.Action {
	return chromedp.Navigate(url)
}

func SelectAction(selector, value string) chromedp.Action {
	return chromedp.SetValue(selector, value, chromedp.ByQuery)
}

func ScreenshotAction(quality int, res *[]byte) chromedp.Action {
	return chromedp.FullScreenshot(res, quality)
}

func WaitVisibleAction(selector string) chromedp.Action {
	return chromedp.WaitVisible(selector, chromedp.ByQuery)
}

func WaitHiddenAction(selector string) chromedp.Action {
	return chromedp.WaitNotVisible(selector, chromedp.ByQuery)
}

func RunScriptAction(script string, res interface{}) chromedp.Action {
	return chromedp.Evaluate(script, res)
}

func ScrollIntoViewAction(selector string) chromedp.Action {
	return chromedp.ScrollIntoView(selector, chromedp.ByQuery)
}

func FocusAction(selector string) chromedp.Action {
	return chromedp.Focus(selector, chromedp.ByQuery)
}

func GetAttributesAction(selector string, res *map[string]string) chromedp.Action {
	return chromedp.Attributes(selector, res, chromedp.ByQuery)
}

// GetNodeIDs returns a slice of nodeIDs for elements matching the selector
func GetNodeIDs(selector string, nodeIDs *[]cdp.NodeID) chromedp.Action {
	return chromedp.NodeIDs(selector, nodeIDs, chromedp.ByQuery)
}

// IsElementPresentAction checks if an element exists without waiting for visibility.
// Moved here from browser/actions.go where it caused an undefined error.
func IsElementPresentAction(selector string, isPresent *bool) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		err := chromedp.Nodes(selector, &nodes, chromedp.ByQuery).Do(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err() // Context cancelled is a real error
			}
			// Other errors often mean "not found" in this context
			*isPresent = false
			return nil // Treat "not found" as success for presence check
		}
		*isPresent = len(nodes) > 0
		return nil
	})
}

// DomNode represents a node in the DOM AST
type DomNode struct {
	NodeType    string              `json:"nodeType"`
	TagName     string              `json:"tagName,omitempty"`
	ID          string              `json:"id,omitempty"`
	Classes     []string            `json:"classes,omitempty"`
	Attributes  map[string]string   `json:"attributes,omitempty"`
	TextContent string              `json:"textContent,omitempty"`
	Children    []DomNode           `json:"children,omitempty"`
}

// GetDomAST generates a DOM AST from the given HTML content
// If parentSelector is provided, it will only generate the AST for that element and its children
// If parentSelector is empty, it will generate the AST for the entire document
func GetDomAST(ctx context.Context, htmlContent, parentSelector string) (*DomNode, error) {
	if htmlContent == "" {
		return nil, fmt.Errorf("empty HTML content")
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// If parentSelector is empty, start from document root
	if parentSelector == "" {
		root := &DomNode{
			NodeType: "document",
			Children: []DomNode{},
		}
		
		// Process the HTML document
		// Process children of the HTML node directly
		for c := doc.FirstChild; c != nil; c = c.NextSibling {
			processNode(c, root)
		}
		return root, nil
	}

	// Otherwise, find the parent node and process from there
	var parentNode *html.Node
	var findParent func(*html.Node)
	
	findParent = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Build a selector for this node to compare
			var id, classes string
			for _, attr := range n.Attr {
				if attr.Key == "id" {
					id = attr.Val
				}
				if attr.Key == "class" {
					classes = attr.Val
				}
			}
			
			// Simple matching based on tag and ID
			if strings.HasPrefix(parentSelector, n.Data) {
				if id != "" && strings.Contains(parentSelector, "#"+id) {
					parentNode = n
					return
				} else if classes != "" {
					// Check if any class in the selector matches
					for _, class := range strings.Fields(classes) {
						if strings.Contains(parentSelector, "."+class) {
							parentNode = n
							return
						}
					}
				} else if parentSelector == n.Data {
					parentNode = n
					return
				}
			}
			
			// Add improved class selector matching (e.g., div.class-name)
			if len(strings.Split(parentSelector, ".")) > 1 {
				parts := strings.Split(parentSelector, ".")
				tagName := parts[0]
				className := parts[1]
				
				// Check if tag name matches and class contains the specified class
				if n.Data == tagName && classes != "" {
					for _, class := range strings.Fields(classes) {
						if class == className || strings.Contains(class, className) {
							parentNode = n
							return
						}
					}
				}
			}
		}
		
		// Recursively check children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if parentNode == nil {
				findParent(c)
			}
		}
	}
	
	findParent(doc)
	
	if parentNode == nil {
		return nil, fmt.Errorf("parent selector '%s' not found", parentSelector)
	}
	
	// Build AST from the found parent node
	root := &DomNode{
		NodeType: "element",
		TagName:  parentNode.Data,
		Children: []DomNode{},
	}
	
	// Process attributes
	processAttributes(parentNode, root)
	
	// Process children
	for c := parentNode.FirstChild; c != nil; c = c.NextSibling {
		processNode(c, root)
	}
	
	return root, nil
}

// processNode recursively processes HTML nodes and builds the DOM AST
func processNode(n *html.Node, parent *DomNode) {
	switch n.Type {
	case html.ElementNode:
		node := DomNode{
			NodeType:   "element",
			TagName:    n.Data,
			Attributes: make(map[string]string),
			Children:   []DomNode{},
		}
		
		// Process attributes
		processAttributes(n, &node)
		
		// Process children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processNode(c, &node)
		}
		
		parent.Children = append(parent.Children, node)
		
	case html.TextNode:
		// Ignore whitespace-only text nodes
		trimmed := strings.TrimSpace(n.Data)
		if trimmed != "" {
			node := DomNode{
				NodeType:    "text",
				TextContent: trimmed,
			}
			parent.Children = append(parent.Children, node)
		}
		
	case html.CommentNode:
		// Optionally include comments
		node := DomNode{
			NodeType:    "comment",
			TextContent: n.Data,
		}
		parent.Children = append(parent.Children, node)
	}
}

// processAttributes extracts attributes from an HTML node
func processAttributes(n *html.Node, node *DomNode) {
	for _, attr := range n.Attr {
		node.Attributes[attr.Key] = attr.Val
		
		// Extract ID and classes for easier access
		if attr.Key == "id" {
			node.ID = attr.Val
		} else if attr.Key == "class" {
			node.Classes = strings.Fields(attr.Val)
		}
	}
}

// GetDomASTAction returns a chromedp action that fetches the DOM AST
func GetDomASTAction(parentSelector string, result *DomNode) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var html string
		
		// First get the HTML content
		if err := chromedp.OuterHTML("html", &html).Do(ctx); err != nil {
			return err
		}
		
		// If there's a parent selector, try to get that element's HTML directly using chromedp
		if parentSelector != "" {
			var parentHTML string
			var exists bool
			
			// Check if the element exists first
			if err := chromedp.Evaluate(fmt.Sprintf(`document.querySelector("%s") !== null`, parentSelector), &exists).Do(ctx); err != nil {
				return err
			}
			
			if !exists {
				return fmt.Errorf("parent selector '%s' not found", parentSelector)
			}
			
			// Get the HTML for that specific element
			if err := chromedp.OuterHTML(parentSelector, &parentHTML).Do(ctx); err != nil {
				return fmt.Errorf("error getting parent element: %w", err)
			}
			
			// Generate AST from the parent HTML
			ast, err := GetDomAST(ctx, parentHTML, "")
			if err != nil {
				return err
			}
			
			// Copy the result
			*result = *ast
			return nil
		}
		
		// If no parent selector, process the full HTML
		ast, err := GetDomAST(ctx, html, "")
		if err != nil {
			return err
		}
		
		// Copy the result
		*result = *ast
		return nil
	})
}

// VerifyChromedpWorkingAction creates an action that tests if chromedp works 
// by visiting a known website and verifying expected elements are present.
// This returns a comprehensive action that checks multiple ChromeDP features.
func VerifyChromedpWorkingAction(result *map[string]interface{}) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var title, html string
		var screenshot []byte
		var isSearchPresent bool

		// Initialize result map if it's nil
		if *result == nil {
			*result = make(map[string]interface{})
		}

		// Create a sequence of actions to verify multiple chromedp features
		err := chromedp.Run(ctx,
			// Navigate to a reliable website for testing
			chromedp.Navigate("https://example.com"),

			// Get page title - basic functionality check
			chromedp.Title(&title),

			// Get page HTML - DOM interaction check
			chromedp.OuterHTML("html", &html),

			// Check if specific element exists - selector functionality check
			IsElementPresentAction("h1", &isSearchPresent),

			// Take a screenshot - demonstrates more complex functionality
			chromedp.CaptureScreenshot(&screenshot),

			// Wait a moment to ensure page is fully loaded
			chromedp.Sleep(1*time.Second),
		)

		// Store test results
		(*result)["title"] = title
		(*result)["html_length"] = len(html)
		(*result)["element_present"] = isSearchPresent
		(*result)["screenshot_size"] = len(screenshot)
		(*result)["success"] = err == nil

		if err != nil {
			(*result)["error"] = err.Error()
			return err
		}

		return nil
	})
}
