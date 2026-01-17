package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

	// Reddit JSON API (better than oEmbed)
	if strings.Contains(urlParam, "reddit.com") {
		if meta, err := h.fetchRedditJSON(urlParam); err == nil {
			json.NewEncoder(w).Encode(meta)
			return
		}
		// Fallback to normal scraping
	}

	// Fetch data
	req, err := http.NewRequest("GET", urlParam, nil)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL"})
		return
	}
	// Use a standard browser UA to avoid 403s (e.g. Wikipedia)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch URL"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "HTTP Error",
			"status": resp.StatusCode,
		})
		return
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse HTML"})
		return
	}

	metadata := make(map[string]string)

	// Helper to extract text from a node's children
	var extractText func(*html.Node) string
	extractText = func(n *html.Node) string {
		var sb strings.Builder
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(extractText(c))
		}
		return sb.String()
	}

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

		// Look for first paragraph if no description yet
		if n.Type == html.ElementNode && n.Data == "p" {
			if _, hasDesc := metadata["description"]; !hasDesc {
				text := strings.TrimSpace(extractText(n))
				// Wikipedia paragraphs often have citations [1] or are empty/short
				// Simple heuristic: length > 50
				if len(text) > 50 {
					// Check for "Coordinates:" which matches length but isn't intro
					if !strings.HasPrefix(text, "Coordinates:") {
						metadata["description"] = text
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// Check for YouTube "soft 404" (Video unavailable)
	// YouTube returns 200 but minimal metadata for unavailable videos.
	if strings.Contains(urlParam, "youtube.com") || strings.Contains(urlParam, "youtu.be") {
		title, hasTitle := metadata["title"]
		// Valid videos usually have a specific title in og:title or title tag.
		// Unavailable videos often have just "- YouTube" or no og:title.
		if !hasTitle || title == " - YouTube" || title == "YouTube" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  "Video Unavailable",
				"status": 404,
			})
			return
		}
	}

	json.NewEncoder(w).Encode(metadata)
}

func (h *Handler) fetchRedditJSON(url string) (map[string]string, error) {
	jsonURL := url + ".json"
	req, err := http.NewRequest("GET", jsonURL, nil)
	if err != nil {
		return nil, err
	}
	// Unique UA to ensure access
	req.Header.Set("User-Agent", "Tumble/1.0 (internal tool; +http://tumble.example.com)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var data []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data")
	}

	meta := make(map[string]string)
	meta["provider_name"] = "Reddit"

	// Traverse: [0] -> data -> children -> [0] -> data
	if listing, ok := data[0].(map[string]interface{}); ok {
		if dataObj, ok := listing["data"].(map[string]interface{}); ok {
			if children, ok := dataObj["children"].([]interface{}); ok && len(children) > 0 {
				if child, ok := children[0].(map[string]interface{}); ok {
					if post, ok := child["data"].(map[string]interface{}); ok {
						if title, ok := post["title"].(string); ok {
							meta["title"] = title
							meta["og:title"] = title
						}

						// Construct description
						author, _ := post["author"].(string)
						sub, _ := post["subreddit_name_prefixed"].(string)
						if author != "" && sub != "" {
							meta["description"] = fmt.Sprintf("Posted by u/%s in %s", author, sub)
						}

						if hint, ok := post["post_hint"].(string); ok {
							if hint == "hosted:video" || hint == "rich:video" {
								meta["type"] = "video"
							}
						}

						// Image extraction
						// 1. Try 'preview' images (highest quality usually)
						foundImage := false
						if preview, ok := post["preview"].(map[string]interface{}); ok {
							if images, ok := preview["images"].([]interface{}); ok && len(images) > 0 {
								if img, ok := images[0].(map[string]interface{}); ok {
									if source, ok := img["source"].(map[string]interface{}); ok {
										if u, ok := source["url"].(string); ok {
											meta["image"] = strings.ReplaceAll(u, "&amp;", "&")
											foundImage = true
										}
									}
								}
							}
						}

						// 2. Fallback to 'thumbnail' if valid URL
						if !foundImage {
							if thumb, ok := post["thumbnail"].(string); ok && strings.HasPrefix(thumb, "http") {
								meta["image"] = thumb
							}
						}
					}
				}
			}
		}
	}

	return meta, nil
}
