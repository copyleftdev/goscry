package dom

import (
	"bytes"
	"context"
	"io"
	"strings"

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
			"p": true, "div": true, "span": true, "br": true, "hr": true,
			"ul": true, "ol": true, "li": true,
			"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "th": true, "td": true,
			"a": true, "button": true, "input": true, "textarea": true, "select": true, "option": true, "label": true,
			"form": true, "img": true, "pre": true, "code": true, "strong": true, "em": true, "b": true, "i": true,
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
