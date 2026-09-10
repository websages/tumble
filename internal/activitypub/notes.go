package activitypub

import (
	"fmt"
	"html"
	"mime"
	"path"
	"strings"
	"time"

	"tumble/internal/data"
)

func (s *Service) linkPermalink(id int) string {
	return fmt.Sprintf("%s/link/%d", s.baseURL(), id)
}

func (s *Service) quotePermalink(id int) string {
	return fmt.Sprintf("%s/quote/%d", s.baseURL(), id)
}

func (s *Service) imagePermalink(id int) string {
	return fmt.Sprintf("%s/image/%d", s.baseURL(), id)
}

// NoteForLink builds the federated Note representation of an IRC link.
func (s *Service) NoteForLink(link *data.IRCLink) *Note {
	permalink := s.linkPermalink(link.ID)
	content := fmt.Sprintf(`<p><a href="%s" rel="nofollow noopener" target="_blank">%s</a></p>`,
		html.EscapeString(link.URL), html.EscapeString(link.Title))
	if link.User != "" {
		content += fmt.Sprintf(`<p>submitted by %s</p>`, html.EscapeString(link.User))
	}
	return &Note{
		ID:           permalink,
		Type:         "Note",
		AttributedTo: s.actorID(),
		Content:      content,
		URL:          permalink,
		Published:    link.Timestamp.UTC().Format(time.RFC3339),
		To:           []string{PublicCollection},
	}
}

// NoteForQuote builds the federated Note representation of a quote.
func (s *Service) NoteForQuote(quote *data.Quote) *Note {
	permalink := s.quotePermalink(quote.ID)
	content := fmt.Sprintf(`<p>%s</p>`, html.EscapeString(quote.Quote))
	if quote.Author != "" {
		content += fmt.Sprintf(`<p>&#8212; %s</p>`, html.EscapeString(quote.Author))
	}
	return &Note{
		ID:           permalink,
		Type:         "Note",
		AttributedTo: s.actorID(),
		Content:      content,
		URL:          permalink,
		Published:    quote.Timestamp.UTC().Format(time.RFC3339),
		To:           []string{PublicCollection},
	}
}

// NoteForImage builds the federated Note representation of an image post.
func (s *Service) NoteForImage(image *data.Image) *Note {
	permalink := s.imagePermalink(image.ID)
	content := fmt.Sprintf(`<p>%s</p>`, html.EscapeString(image.Title))
	return &Note{
		ID:           permalink,
		Type:         "Note",
		AttributedTo: s.actorID(),
		Content:      content,
		URL:          permalink,
		Published:    image.Timestamp.UTC().Format(time.RFC3339),
		To:           []string{PublicCollection},
		Attachment: []Attachment{
			{Type: "Image", MediaType: guessImageMediaType(image.URL), URL: image.URL},
		},
	}
}

func guessImageMediaType(imageURL string) string {
	ext := strings.ToLower(path.Ext(imageURL))
	if mt := mime.TypeByExtension(ext); mt != "" {
		return mt
	}
	return "image/jpeg"
}
