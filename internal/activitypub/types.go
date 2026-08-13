package activitypub

// ContextURL is the standard ActivityStreams JSON-LD context.
const ContextURL = "https://www.w3.org/ns/activitystreams"

// SecurityContextURL adds the publicKey vocabulary used by HTTP Signatures.
const SecurityContextURL = "https://w3id.org/security/v1"

// PublicCollection is the well-known "public" audience URI.
const PublicCollection = "https://www.w3.org/ns/activitystreams#Public"

// Actor represents the site-wide ActivityPub actor (a Service, since the
// site has no per-user accounts).
type Actor struct {
	Context           []string   `json:"@context"`
	ID                string     `json:"id"`
	Type              string     `json:"type"`
	PreferredUsername string     `json:"preferredUsername"`
	Name              string     `json:"name,omitempty"`
	Summary           string     `json:"summary,omitempty"`
	Inbox             string     `json:"inbox"`
	Outbox            string     `json:"outbox"`
	Followers         string     `json:"followers"`
	URL               string     `json:"url,omitempty"`
	PublicKey         PublicKey  `json:"publicKey"`
	Endpoints         *Endpoints `json:"endpoints,omitempty"`
}

// Endpoints advertises a shared inbox, if the remote actor's server supports
// delivering a single copy of an activity to multiple local followers.
type Endpoints struct {
	SharedInbox string `json:"sharedInbox,omitempty"`
}

// PublicKey is the actor's RSA public key, used by remote servers to verify
// HTTP Signatures on activities delivered from this site.
type PublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

// WebfingerResponse is the JRD document returned by /.well-known/webfinger.
type WebfingerResponse struct {
	Subject string          `json:"subject"`
	Links   []WebfingerLink `json:"links"`
}

type WebfingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

// OrderedCollection is the top-level (unpaginated) view of a collection,
// pointing at the first page.
type OrderedCollection struct {
	Context    string `json:"@context"`
	ID         string `json:"id"`
	Type       string `json:"type"`
	TotalItems int    `json:"totalItems"`
	First      string `json:"first,omitempty"`
}

// OrderedCollectionPage is a single page of items within a collection.
type OrderedCollectionPage struct {
	Context      string `json:"@context"`
	ID           string `json:"id"`
	Type         string `json:"type"`
	PartOf       string `json:"partOf"`
	Next         string `json:"next,omitempty"`
	OrderedItems []any  `json:"orderedItems"`
}

// Activity is a generic outbound ActivityStreams activity (Create, Accept, ...).
type Activity struct {
	Context   any      `json:"@context,omitempty"`
	ID        string   `json:"id,omitempty"`
	Type      string   `json:"type"`
	Actor     string   `json:"actor,omitempty"`
	Object    any      `json:"object,omitempty"`
	To        []string `json:"to,omitempty"`
	Published string   `json:"published,omitempty"`
}

// Note represents a federated post (link, quote, or image).
type Note struct {
	Context      any          `json:"@context,omitempty"`
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	AttributedTo string       `json:"attributedTo"`
	Content      string       `json:"content"`
	URL          string       `json:"url"`
	Published    string       `json:"published"`
	To           []string     `json:"to,omitempty"`
	Attachment   []Attachment `json:"attachment,omitempty"`
}

// Attachment is a media attachment on a Note (used for images).
type Attachment struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType,omitempty"`
	URL       string `json:"url"`
}

// IncomingActivity is a loosely-typed inbound activity, used to inspect the
// type/actor before deciding how (or whether) to handle it.
type IncomingActivity struct {
	Context any    `json:"@context,omitempty"`
	ID      string `json:"id"`
	Type    string `json:"type"`
	Actor   string `json:"actor"`
	Object  any    `json:"object"`
}
