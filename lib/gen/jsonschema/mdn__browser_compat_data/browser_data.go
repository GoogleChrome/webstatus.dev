// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    browserData, err := UnmarshalBrowserData(bytes)
//    bytes, err = browserData.Marshal()

package mdn__browser_compat_data

import "encoding/json"

func UnmarshalBrowserData(data []byte) (BrowserData, error) {
	var r BrowserData
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *BrowserData) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type BrowserData struct {
	Browsers map[string]BrowserStatement `json:"browsers,omitempty"`
}

type BrowserStatement struct {
	// Whether the browser supports user-toggleable flags that enable or disable features.                                   
	AcceptsFlags                                                                                 bool                        `json:"accepts_flags"`
	// Whether the browser supports extensions.                                                                              
	AcceptsWebextensions                                                                         bool                        `json:"accepts_webextensions"`
	// The browser brand name (e.g. Firefox, Firefox Android, Chrome, etc.).                                                 
	Name                                                                                         string                      `json:"name"`
	// URL of the page where feature flags can be changed (e.g. 'about:config' for Firefox or                                
	// 'chrome://flags' for Chrome).                                                                                         
	PrefURL                                                                                      *string                     `json:"pref_url,omitempty"`
	// The name of the browser's preview channel (e.g. 'Nightly' for Firefox or 'TP' for Safari).                            
	PreviewName                                                                                  *string                     `json:"preview_name,omitempty"`
	// The known versions of this browser.                                                                                   
	Releases                                                                                     map[string]ReleaseStatement `json:"releases"`
	// The platform the browser runs on (e.g. desktop, mobile, XR, or server engine).                                        
	Type                                                                                         BrowserType                 `json:"type"`
	// The upstream browser this browser derives from (e.g. Firefox Android is derived from                                  
	// Firefox, Edge is derived from Chrome).                                                                                
	Upstream                                                                                     *string                     `json:"upstream,omitempty"`
}

type ReleaseStatement struct {
	// Name of the browser's underlying engine.                                                          
	Engine                                                                                *BrowserEngine `json:"engine,omitempty"`
	// Version of the engine corresponding to the browser version.                                       
	EngineVersion                                                                         *string        `json:"engine_version,omitempty"`
	// The date on which this version was released, formatted as `YYYY-MM-DD`.                           
	ReleaseDate                                                                           *string        `json:"release_date,omitempty"`
	// A link to the release notes or changelog for a given release.                                     
	ReleaseNotes                                                                          *string        `json:"release_notes,omitempty"`
	// A property indicating where in the lifetime cycle this release is in (e.g. current,               
	// retired, beta, nightly).                                                                          
	Status                                                                                BrowserStatus  `json:"status"`
}

// Name of the browser's underlying engine.
type BrowserEngine string

const (
	Blink    BrowserEngine = "Blink"
	EdgeHTML BrowserEngine = "EdgeHTML"
	Gecko    BrowserEngine = "Gecko"
	Presto   BrowserEngine = "Presto"
	Trident  BrowserEngine = "Trident"
	V8       BrowserEngine = "V8"
	WebKit   BrowserEngine = "WebKit"
)

// A property indicating where in the lifetime cycle this release is in (e.g. current,
// retired, beta, nightly).
type BrowserStatus string

const (
	Beta    BrowserStatus = "beta"
	Current BrowserStatus = "current"
	ESR     BrowserStatus = "esr"
	Nightly BrowserStatus = "nightly"
	Planned BrowserStatus = "planned"
	Retired BrowserStatus = "retired"
)

// The platform the browser runs on (e.g. desktop, mobile, XR, or server engine).
type BrowserType string

const (
	Desktop BrowserType = "desktop"
	Mobile  BrowserType = "mobile"
	Server  BrowserType = "server"
	Xr      BrowserType = "xr"
)
