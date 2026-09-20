// Package judge holds the domain: question-packs (the rules for a surface) and the
// logic that runs a pack over items and ranks the result.
package judge

import "jevai/internal/jev"

// Check is one weighted, atomic question plus how its answer maps to risk.
type Check struct {
	Key         string
	Label       string // human label for the UI
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
		{Key: "title_mismatch", Label: "Title–description mismatch", Weight: 1.0, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does the title describe a DIFFERENT product than the description? Answer true only if they clearly conflict.",
		}},
		{Key: "wrong_category", Label: "Wrong category", Weight: 0.8, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Is the stated category wrong for the product described?",
		}},
		{Key: "prohibited_claim", Label: "Prohibited claim", Weight: 1.2, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does the copy make a prohibited or overreaching claim (medical cure, guaranteed results, counterfeit or brand misuse)?",
		}},
		{Key: "missing_detail", Label: "Missing detail", Weight: 0.5, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Is the description missing basic detail a buyer needs (size, material, quantity, or condition)?",
		}},
	},
}

// ClaimScreening flags regulated-marketing risks in a piece of copy. The rules are
// ILLUSTRATIVE and UNVERIFIED — the UI states this and that it is not legal advice
// (PRD §7 / workspace provenance rule). `category` is provided in the state for context.
var ClaimScreening = Pack{
	Name: "claim-screening",
	Checks: []Check{
		{Key: "disease_claim", Label: "Treats/cures a disease", Weight: 1.3, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `copy` claim to treat, cure, or prevent a disease or medical condition?",
		}},
		{Key: "guaranteed_result", Label: "Guaranteed / unsubstantiated result", Weight: 1.0, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `copy` promise a guaranteed or unsubstantiated result (e.g. 'lose 10kg in a week', '100% effective')?",
		}},
		{Key: "certification_asserted", Label: "Asserts a certification", Weight: 0.9, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `copy` assert an official certification as fact (halal, BPOM-registered, clinically proven, doctor-approved)?",
		}},
		{Key: "restricted_ingredient", Label: "Restricted ingredient", Weight: 1.1, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `copy` mention an ingredient commonly restricted or banned in this product category?",
		}},
		{Key: "misleading_comparison", Label: "Misleading comparison", Weight: 0.7, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `copy` make a misleading superiority, 'best', or before/after claim it cannot substantiate?",
		}},
	},
}

// AdPreflight screens an ad creative before spend. The landing_mismatch check reads
// the creative against `landing` (pasted landing-page copy) when provided.
var AdPreflight = Pack{
	Name: "ad-preflight",
	Checks: []Check{
		{Key: "prohibited_claim", Label: "Prohibited / overreaching claim", Weight: 1.2, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `creative` make a prohibited or overreaching advertising claim (medical cure, guaranteed income or results, miracle outcome)?",
		}},
		{Key: "brand_unsafe", Label: "Brand-unsafe content", Weight: 1.0, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `creative` contain brand-unsafe content (adult, violence, hate, shocking, or illegal)?",
		}},
		{Key: "restricted_category", Label: "Restricted ad category", Weight: 0.8, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `creative` fall in a restricted ad category (health, finance, politics, dating) that needs special rules for `platform`?",
		}},
		{Key: "misleading_urgency", Label: "Misleading urgency / superlative", Weight: 0.7, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Does `creative` use misleading urgency, fake scarcity, or unsupported superlatives ('#1', 'best ever')?",
		}},
		{Key: "landing_mismatch", Label: "Creative ≠ landing page", Weight: 1.0, BadWhenTrue: true, Question: jev.Question{
			Type:         "noul",
			Instructions: "Given `landing`, does the ad `creative` promise something the landing page does not deliver? Answer false if `landing` is empty.",
		}},
	},
}
