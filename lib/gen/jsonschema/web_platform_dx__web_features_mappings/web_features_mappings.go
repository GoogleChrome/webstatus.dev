// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    webFeaturesMappings, err := UnmarshalWebFeaturesMappings(bytes)
//    bytes, err = webFeaturesMappings.Marshal()

package web_platform_dx__web_features_mappings

import "encoding/json"

type WebFeaturesMappings map[string]WebFeaturesMapping

func UnmarshalWebFeaturesMappings(data []byte) (WebFeaturesMappings, error) {
	var r WebFeaturesMappings
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *WebFeaturesMappings) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type WebFeaturesMapping struct {
	ChromeUseCounters  *UseCounterValue          `json:"chrome-use-counters,omitempty"`
	DeveloperSignals   *DeveloperSignal          `json:"developer-signals,omitempty"`
	Interop            []InteropRecord           `json:"interop,omitempty"`
	MdnDocs            []MdnLink                 `json:"mdn-docs,omitempty"`
	StandardsPositions []StandardsPositionRecord `json:"standards-positions,omitempty"`
	StateOfSurveys     []SurveyRecord            `json:"state-of-surveys,omitempty"`
	Wpt                *WptRecord                `json:"wpt,omitempty"`
}

// The usage metrics for a single web feature from chrome-use-counters.json.
type UseCounterValue struct {
	PercentageOfPageLoad float64 `json:"percentageOfPageLoad"`
	URL                  string  `json:"url"`
}

// The developer signals for a single web feature from developer-signals.json.
type DeveloperSignal struct {
	URL   string `json:"url"`
	Votes int64  `json:"votes"`
}

// An array of Interop records from interop.json.
type InteropRecord struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Year  int64  `json:"year"`
}

// An array of MDN documentation links from mdn-docs.json.
type MdnLink struct {
	Anchor *string `json:"anchor"`
	Slug   string  `json:"slug"`
	Title  string  `json:"title"`
	URL    string  `json:"url"`
}

// An array of standards position records from standards-positions.json.
type StandardsPositionRecord struct {
	Concerns []Concern `json:"concerns"`
	Position Position  `json:"position"`
	URL      string    `json:"url"`
	Vendor   Vendor    `json:"vendor"`
}

// An array of survey records from state-of-surveys.json.
type SurveyRecord struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Question    string `json:"question"`
	SubQuestion string `json:"subQuestion"`
	URL         string `json:"url"`
}

// A record of Web Platform Tests from wpt.json.
type WptRecord struct {
	URL string `json:"url"`
}

type Concern string

const (
	APIDesign            Concern = "API design"
	Accessibility        Concern = "accessibility"
	Annoyance            Concern = "annoyance"
	Compatibility        Concern = "compatibility"
	Complexity           Concern = "complexity"
	Dependencies         Concern = "dependencies"
	DeviceIndependence   Concern = "device independence"
	Duplication          Concern = "duplication"
	Integration          Concern = "integration"
	Internationalization Concern = "internationalization"
	Interoperability     Concern = "interoperability"
	Maintenance          Concern = "maintenance"
	Performance          Concern = "performance"
	Portability          Concern = "portability"
	Power                Concern = "power"
	Privacy              Concern = "privacy"
	Security             Concern = "security"
	UseCases             Concern = "use cases"
	Venue                Concern = "venue"
)

type Position string

const (
	Blocked  Position = "blocked"
	Defer    Position = "defer"
	Empty    Position = ""
	Negative Position = "negative"
	Neutral  Position = "neutral"
	Oppose   Position = "oppose"
	Positive Position = "positive"
	Support  Position = "support"
)

type Vendor string

const (
	Apple   Vendor = "apple"
	Mozilla Vendor = "mozilla"
)
