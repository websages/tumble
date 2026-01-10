package handler

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/html"
)

// OGPreviewHandler handles /ogpreview.cgi
func (h *Handler) OGPreviewHandler(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	w.Header().Set("Content-Type", "application/json")

	if urlParam == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "No URL provided"})
		return
	}

	// Fetch data
	resp, err := http.Get(urlParam)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch URL"})
		return
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse HTML"})
		return
	}

	metadata := make(map[string]string)
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content, name string
			for _, a := range n.Attr {
				if a.Key == "property" {
					property = a.Val
				}
				if a.Key == "content" {
					content = a.Val
				}
				if a.Key == "name" {
					name = a.Val
				}
			}

			if property == "og:title" {
				metadata["title"] = content
			} else if property == "og:description" {
				metadata["description"] = content
			} else if property == "og:image" {
				metadata["image"] = content
			} else if name == "twitter:image" {
				metadata["twitter_image"] = content
			} else if name == "twitter:title" {
				metadata["twitter_title"] = content
			} else if name == "twitter:description" {
				metadata["twitter_description"] = content
			} else if name == "description" {
				if _, ok := metadata["description"]; !ok {
					metadata["description"] = content
				}
			}
		}
		// Also look for title tag
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				if _, ok := metadata["title"]; !ok {
					metadata["title"] = n.FirstChild.Data
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	json.NewEncoder(w).Encode(metadata)
}
