// Package judge holds reusable question-packs — the domain rules for each surface,
// expressed as narrow atomic Jev questions (see PRD §5 design rules).
package judge

import "jevai/internal/jev"

// Pack is a named set of typed questions applied to one item.
type Pack struct {
	Name      string
	Questions map[string]jev.Question
}

// ListingHygiene is an illustrative first pack (expand per the chosen surface, PRD §7).
// Each question is narrow and atomic so answers can be inspected and weighted in code.
var ListingHygiene = Pack{
	Name: "listing-hygiene",
	Questions: map[string]jev.Question{
		"title_matches_desc": {
			Type:         "noul",
			Instructions: "Does the title accurately describe the product in the description?",
		},
		"category_correct": {
			Type:         "noul",
			Instructions: "Is the stated category correct for this product?",
		},
		"prohibited_claim": {
			Type:         "noul",
			Instructions: "Does the copy make a prohibited or overreaching claim?",
		},
	},
}
