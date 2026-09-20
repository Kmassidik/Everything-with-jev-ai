// Package judge holds the domain: question-packs (the rules for a surface) and the
// logic that runs a pack over many items and ranks the result.
package judge

import "jevai/internal/jev"

// Check is one weighted, atomic question plus how its answer maps to risk.
type Check struct {
	Key         string
	Question    jev.Question
	Weight      float64
	BadWhenTrue bool // true: a high probability means risk; false: a low probability means risk
}

// Pack is a named set of checks applied to one item.
type Pack struct {
	Name   string
	Checks []Check
}

// questions builds the map sent to Jev in a single request.
func (p Pack) questions() map[string]jev.Question {
	m := make(map[string]jev.Question, len(p.Checks))
	for _, c := range p.Checks {
		m[c.Key] = c.Question
	}
	return m
}

// ListingHygiene screens a marketplace listing. Each question is narrow and phrased
// so that "true" is the problem (PRD §5: atomic questions, uniform polarity).
var ListingHygiene = Pack{
	Name: "listing-hygiene",
	Checks: []Check{
		{Key: "title_mismatch", Weight: 1.0, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does the title describe a DIFFERENT product than the description? Answer true only if they clearly conflict.",
		}},
		{Key: "wrong_category", Weight: 0.8, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Is the stated category wrong for the product described?",
		}},
		{Key: "prohibited_claim", Weight: 1.2, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does the copy make a prohibited or overreaching claim (medical cure, guaranteed results, counterfeit or brand misuse)?",
		}},
		{Key: "missing_detail", Weight: 0.5, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Is the description missing basic detail a buyer needs (size, material, quantity, or condition)?",
		}},
	},
}
