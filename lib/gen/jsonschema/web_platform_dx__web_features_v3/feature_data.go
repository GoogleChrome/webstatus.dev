// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    featureData, err := UnmarshalFeatureData(bytes)
//    bytes, err = featureData.Marshal()

package web_platform_dx__web_features_v3

import "bytes"
import "errors"

import "encoding/json"

func UnmarshalFeatureData(data []byte) (FeatureData, error) {
	var r FeatureData
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FeatureData) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// The top-level web-features data package
type FeatureData struct {
	// Browsers and browser release data                        
	Browsers                            Browsers                `json:"browsers"`
	// Feature identifiers and data                             
	Features                            map[string]FeatureValue `json:"features"`
	// Group identifiers and data                               
	Groups                              map[string]GroupData    `json:"groups"`
	// Snapshot identifiers and data                            
	Snapshots                           map[string]SnapshotData `json:"snapshots"`
}

// Browsers and browser release data
type Browsers struct {
	Chrome         BrowserData `json:"chrome"`
	ChromeAndroid  BrowserData `json:"chrome_android"`
	Edge           BrowserData `json:"edge"`
	Firefox        BrowserData `json:"firefox"`
	FirefoxAndroid BrowserData `json:"firefox_android"`
	Safari         BrowserData `json:"safari"`
	SafariIos      BrowserData `json:"safari_ios"`
}

// Browser information
type BrowserData struct {
	// The name of the browser, as in "Edge" or "Safari on iOS"          
	Name                                                       string    `json:"name"`
	Releases                                                   []Release `json:"releases"`
}

// Browser release information
type Release struct {
	// The release date, as in "2023-12-11"           
	Date                                       string `json:"date"`
	// The version string, as in "10" or "17.1"       
	Version                                    string `json:"version"`
}

// A feature data entry
//
// A feature has permanently moved to exactly one other ID
//
// A feature has split into two or more other features
type FeatureValue struct {
	// caniuse.com identifiers                                                                               
	Caniuse                                                                                  []string        `json:"caniuse,omitempty"`
	// Sources of support data for this feature                                                              
	CompatFeatures                                                                           []string        `json:"compat_features,omitempty"`
	// Short description of the feature, as a plain text string                                              
	Description                                                                              *string         `json:"description,omitempty"`
	// Short description of the feature, as an HTML string                                                   
	DescriptionHTML                                                                          *string         `json:"description_html,omitempty"`
	// Whether developers are formally discouraged from using this feature                                   
	Discouraged                                                                              *Discouraged    `json:"discouraged,omitempty"`
	// Group identifiers                                                                                     
	Group                                                                                    []string        `json:"group,omitempty"`
	Kind                                                                                     Kind            `json:"kind"`
	// Short name                                                                                            
	Name                                                                                     *string         `json:"name,omitempty"`
	// Snapshot identifiers                                                                                  
	Snapshot                                                                                 []string        `json:"snapshot,omitempty"`
	// Specification URLs                                                                                    
	Spec                                                                                     []string        `json:"spec,omitempty"`
	// Whether a feature is considered a "Baseline" web platform feature and when it achieved                
	// that status                                                                                           
	Status                                                                                   *StatusHeadline `json:"status,omitempty"`
	// The new ID for this feature                                                                           
	RedirectTarget                                                                           *string         `json:"redirect_target,omitempty"`
	// The new IDs for this feature                                                                          
	RedirectTargets                                                                          []string        `json:"redirect_targets,omitempty"`
}

// Whether developers are formally discouraged from using this feature
type Discouraged struct {
	// Links to a formal discouragement notice, such as specification text, intent-to-unship,         
	// etc.                                                                                           
	AccordingTo                                                                              []string `json:"according_to"`
	// IDs for features that substitute some or all of this feature's utility                         
	Alternatives                                                                             []string `json:"alternatives,omitempty"`
}

// Whether a feature is considered a "Baseline" web platform feature and when it achieved
// that status
type StatusHeadline struct {
	// Whether the feature is Baseline (low substatus), Baseline (high substatus), or not (false)                  
	Baseline                                                                                     *BaselineUnion    `json:"baseline"`
	// Date the feature achieved Baseline high status                                                              
	BaselineHighDate                                                                             *string           `json:"baseline_high_date,omitempty"`
	// Date the feature achieved Baseline low status                                                               
	BaselineLowDate                                                                              *string           `json:"baseline_low_date,omitempty"`
	// Statuses for each key in the feature's compat_features list, if applicable.                                 
	ByCompatKey                                                                                  map[string]Status `json:"by_compat_key,omitempty"`
	// Browser versions that most-recently introduced the feature                                                  
	Support                                                                                      Support           `json:"support"`
}

type Status struct {
	// Whether the feature is Baseline (low substatus), Baseline (high substatus), or not (false)               
	Baseline                                                                                     *BaselineUnion `json:"baseline"`
	// Date the feature achieved Baseline high status                                                           
	BaselineHighDate                                                                             *string        `json:"baseline_high_date,omitempty"`
	// Date the feature achieved Baseline low status                                                            
	BaselineLowDate                                                                              *string        `json:"baseline_low_date,omitempty"`
	// Browser versions that most-recently introduced the feature                                               
	Support                                                                                      Support        `json:"support"`
}

// Browser versions that most-recently introduced the feature
type Support struct {
	Chrome         *string `json:"chrome,omitempty"`
	ChromeAndroid  *string `json:"chrome_android,omitempty"`
	Edge           *string `json:"edge,omitempty"`
	Firefox        *string `json:"firefox,omitempty"`
	FirefoxAndroid *string `json:"firefox_android,omitempty"`
	Safari         *string `json:"safari,omitempty"`
	SafariIos      *string `json:"safari_ios,omitempty"`
}

type GroupData struct {
	// Short name                        
	Name                         string  `json:"name"`
	// Identifier of parent group        
	Parent                       *string `json:"parent,omitempty"`
}

type SnapshotData struct {
	// Short name          
	Name            string `json:"name"`
	// Specification       
	Spec            string `json:"spec"`
}

type Kind string

const (
	Feature Kind = "feature"
	Moved   Kind = "moved"
	Split   Kind = "split"
)

type BaselineEnum string

const (
	High BaselineEnum = "high"
	Low  BaselineEnum = "low"
)

// Whether the feature is Baseline (low substatus), Baseline (high substatus), or not (false)
type BaselineUnion struct {
	Bool *bool
	Enum *BaselineEnum
}

func (x *BaselineUnion) UnmarshalJSON(data []byte) error {
	x.Enum = nil
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, false, nil, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *BaselineUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, false, nil, false, nil, x.Enum != nil, x.Enum, false)
}

func unmarshalUnion(data []byte, pi **int64, pf **float64, pb **bool, ps **string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) (bool, error) {
	if pi != nil {
			*pi = nil
	}
	if pf != nil {
			*pf = nil
	}
	if pb != nil {
			*pb = nil
	}
	if ps != nil {
			*ps = nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
			return false, err
	}

	switch v := tok.(type) {
	case json.Number:
			if pi != nil {
					i, err := v.Int64()
					if err == nil {
							*pi = &i
							return false, nil
					}
			}
			if pf != nil {
					f, err := v.Float64()
					if err == nil {
							*pf = &f
							return false, nil
					}
					return false, errors.New("Unparsable number")
			}
			return false, errors.New("Union does not contain number")
	case float64:
			return false, errors.New("Decoder should not return float64")
	case bool:
			if pb != nil {
					*pb = &v
					return false, nil
			}
			return false, errors.New("Union does not contain bool")
	case string:
			if haveEnum {
					return false, json.Unmarshal(data, pe)
			}
			if ps != nil {
					*ps = &v
					return false, nil
			}
			return false, errors.New("Union does not contain string")
	case nil:
			if nullable {
					return false, nil
			}
			return false, errors.New("Union does not contain null")
	case json.Delim:
			if v == '{' {
					if haveObject {
							return true, json.Unmarshal(data, pc)
					}
					if haveMap {
							return false, json.Unmarshal(data, pm)
					}
					return false, errors.New("Union does not contain object")
			}
			if v == '[' {
					if haveArray {
							return false, json.Unmarshal(data, pa)
					}
					return false, errors.New("Union does not contain array")
			}
			return false, errors.New("Cannot handle delimiter")
	}
	return false, errors.New("Cannot unmarshal union")
}

func marshalUnion(pi *int64, pf *float64, pb *bool, ps *string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) ([]byte, error) {
	if pi != nil {
			return json.Marshal(*pi)
	}
	if pf != nil {
			return json.Marshal(*pf)
	}
	if pb != nil {
			return json.Marshal(*pb)
	}
	if ps != nil {
			return json.Marshal(*ps)
	}
	if haveArray {
			return json.Marshal(pa)
	}
	if haveObject {
			return json.Marshal(pc)
	}
	if haveMap {
			return json.Marshal(pm)
	}
	if haveEnum {
			return json.Marshal(pe)
	}
	if nullable {
			return json.Marshal(nil)
	}
	return nil, errors.New("Union must not be null")
}
