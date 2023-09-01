// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    featureData, err := UnmarshalFeatureData(bytes)
//    bytes, err = featureData.Marshal()

package web_platform_dx__web_features

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

// Web platform feature
type FeatureData struct {
	// Alias identifier                                                                               
	Alias                                                                                    *Alias   `json:"alias"`
	// caniuse.com identifier                                                                         
	Caniuse                                                                                  *string  `json:"caniuse,omitempty"`
	// Sources of support data for this feature                                                       
	CompatFeatures                                                                           []string `json:"compat_features,omitempty"`
	// Specification                                                                                  
	Spec                                                                                     *Alias   `json:"spec"`
	// Whether a feature is considered a "baseline" web platform feature and when it achieved         
	// that status                                                                                    
	Status                                                                                   *Status  `json:"status,omitempty"`
	// Usage stats                                                                                    
	UsageStats                                                                               *Alias   `json:"usage_stats"`
}

// Whether a feature is considered a "baseline" web platform feature and when it achieved
// that status
type Status struct {
	// Whether the feature achieved baseline status                       
	IsBaseline                                                   bool     `json:"is_baseline"`
	// Date the feature achieved baseline status                          
	Since                                                        *string  `json:"since,omitempty"`
	// Browser versions that most-recently introduced the feature         
	Support                                                      *Support `json:"support,omitempty"`
}

// Browser versions that most-recently introduced the feature
type Support struct {
	Chrome  *string `json:"chrome,omitempty"`
	Edge    *string `json:"edge,omitempty"`
	Firefox *string `json:"firefox,omitempty"`
	Safari  *string `json:"safari,omitempty"`
}

// Alias identifier
//
// Specification
//
// Usage stats
type Alias struct {
	String      *string
	StringArray []string
}

func (x *Alias) UnmarshalJSON(data []byte) error {
	x.StringArray = nil
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, true, &x.StringArray, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *Alias) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, x.StringArray != nil, x.StringArray, false, nil, false, nil, false, nil, false)
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
