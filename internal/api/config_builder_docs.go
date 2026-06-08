package api

import "github.com/MattCheramie/GopherTrunk/internal/configbuilder"

// DocLink carries a config section's shared help for the web Config Builder:
// the section Instructions plus a documentation link (Title/URL) opened in a
// new tab from the section's help affordance. Both come from the single Go
// source of truth in internal/configbuilder (Sections), so the web and the
// terminal builder present identical section help.
type DocLink struct {
	Title        string `json:"title"`
	URL          string `json:"url"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
}

// configDocs projects configbuilder.Sections() into the section-keyed DocLink
// map the web builder fetches over GET /api/v1/config/docs.
func configDocs() map[string]DocLink {
	out := make(map[string]DocLink, len(configbuilder.Sections()))
	for _, s := range configbuilder.Sections() {
		out[s.Key] = DocLink{
			Title:        s.DocTitle,
			URL:          s.DocURL,
			Description:  s.Instructions,
			Instructions: s.Instructions,
		}
	}
	return out
}
