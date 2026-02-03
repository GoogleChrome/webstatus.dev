// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// nolint:lll  // WONTFIX - for readability
package digest

const componentStyles = `{{- define "style_badge_wrapper" -}}align-self: stretch; padding-top: 12px; padding-bottom: 11px; padding-left: 15px; padding-right: 16px; overflow: hidden; border-top-left-radius: 4px; border-top-right-radius: 4px; justify-content: flex-start; align-items: center; display: flex;{{- end -}}
{{- define "style_badge_inner_wrapper" -}}flex: 1 1 0; flex-direction: column; justify-content: center; align-items: flex-start; display: inline-flex;{{- end -}}
{{- define "style_change_detail_wrapper" -}}align-self: stretch; justify-content: flex-start; align-items: center; gap: 10px; display: inline-flex; width: 100%;{{- end -}}
{{- define "style_change_detail_inner" -}}flex: 1 1 0;{{- end -}}
{{- define "style_banner_wrapper" -}}align-self: stretch; height: 50px; padding-top: 12px; padding-bottom: 11px; padding-left: 15px; padding-right: 16px; overflow: hidden; border-top-left-radius: 4px; border-top-right-radius: 4px; justify-content: flex-start; align-items: center; gap: 8px; display: flex;{{- end -}}
{{- define "style_banner_icon_wrapper_28" -}}height: 28px; position: relative; overflow: hidden; display: flex; align-items: center;{{- end -}}
{{- define "style_banner_icon_wrapper_20" -}}height: 20px; position: relative; margin-right: 4px;{{- end -}}
{{- define "style_img_responsive" -}}display: block; width: auto;{{- end -}}
{{- define "style_banner_text_wrapper" -}}flex: 1 1 0;{{- end -}}
{{- define "style_banner_browser_logos_wrapper" -}}justify-content: flex-start; align-items: center; display: flex; margin-right: 8px;{{- end -}}
{{- define "style_browser_item_row" -}}align-self: stretch; justify-content: flex-start; align-items: center; gap: 10px; display: flex;{{- end -}}
{{- define "style_browser_item_logo_wrapper" -}}justify-content: flex-start; align-items: center; display: flex;{{- end -}}
{{- define "style_browser_item_feature_link_wrapper" -}}align-self: stretch; justify-content: flex-start; align-items: center; gap: 10px; display: inline-flex; margin-top: 8px;{{- end -}}
{{- define "style_button_wrapper" -}}margin: 20px 0; text-align: center;{{- end -}}
{{- define "style_footer_wrapper" -}}align-self: stretch; padding-top: 16px; flex-direction: column; justify-content: flex-start; align-items: flex-start; gap: 12px; display: flex; {{- template "font_family_main" -}};{{- end -}}
{{- define "style_footer_hr" -}}align-self: stretch; height: 1px; background: #E4E4E7;{{- end -}}
{{- define "style_footer_text_wrapper" -}}align-self: stretch;{{- end -}}
{{- define "style_feature_title_row_wrapper" -}}align-self: stretch; justify-content: flex-start; align-items: center; gap: 10px; display: flex;{{- end -}}
{{- define "style_feature_title_row_inner" -}}flex: 1 1 0;{{- end -}}`

const badgeComponent = `{{- define "badge" -}}
<div style='{{- template "style_badge_wrapper" -}}; background: {{- badgeBackgroundColor .Title -}};'>
    <div style='{{- template "style_badge_inner_wrapper" -}}'>
        <div style='{{- template "style_text_badge_title" -}}'>{{.Title}}</div>
        {{- if .Description -}}
        <div style='{{- template "style_text_badge_description" -}}'>{{.Description}}</div>
        {{- end -}}
    </div>
</div>
{{- end -}}`

const introTextComponent = `{{- define "intro_text" -}}
<div style='{{- template "style_section_wrapper" -}}'>
    <h2 style='{{- template "style_subject_header" -}}'>{{.Subject}}</h2>
    <div style='{{- template "style_query_text" -}}'>
        Here is your update for the saved search <strong style='font-weight: bold;'>'{{.Query}}'</strong>. 
        {{.SummaryText}}.
    </div>
</div>
{{- end -}}`

const changeDetailComponent = `{{- define "change_detail" -}}
<div style='{{- template "style_change_detail_wrapper" -}}'>
    <div style='{{- template "style_change_detail_inner" -}}'>
        <span style='{{- template "style_text_body" -}}'>{{.Label}}</span>
        <span style='{{- template "style_text_body_subtle" -}}'> ({{.From}} &rarr; {{.To}})</span>
    </div>
</div>
{{- end -}}`

const baselineChangeItemComponent = `{{- define "baseline_change_item" -}}
<div style='{{- template "style_section_wrapper" -}}'>
    <div style='{{- template "style_banner_wrapper" -}}; {{- template "color_bg_success" -}}'>
        <div style='{{- template "style_banner_icon_wrapper_28" -}}'>
            <img src="{{.ToURL}}" height="28" alt="{{.To}}" style='{{- template "style_img_responsive" -}}' />
        </div>
        <div style='{{- template "style_banner_text_wrapper" -}}'>
            <span style='{{- template "style_text_banner_bold" -}}'>Baseline</span>
            <span style='{{- template "style_text_banner_normal" -}}'> {{.To}} </span>
        </div>
    </div>
    <div style='{{- template "style_card_body" -}}'>
        <div style='{{- template "style_browser_item_row" -}}'>
            <div style='{{- template "style_banner_text_wrapper" -}}'>
                <span style='{{- template "style_text_feature_link" -}}'>{{.FeatureName}}</span>
            </div>
            <!-- Optional Date logic could go here if passed -->
        </div>
    </div>
</div>
{{- end -}}`

const browserItemComponent = `{{- define "browser_item" -}}
<div style='{{- template "style_card_body" -}}'>
    <div style='{{- template "style_browser_item_row" -}}'>
        <div style='{{- template "style_browser_item_logo_wrapper" -}}'>
            <img src="{{.LogoURL}}" height="20" alt="{{.Name}}" style='{{- template "style_img_responsive" -}}' />
        </div>
        <div style='{{- template "style_text_browser_item" -}}'>
            {{.Name}}: {{ template "browser_status_detail" .From }} &rarr; {{ template "browser_status_detail" .To -}}
        </div>
    </div>
    {{- if .FeatureName -}}
        <div style='{{- template "style_browser_item_feature_link_wrapper" -}}'>
            <div style='{{- template "style_banner_text_wrapper" -}}'>
                <a href="{{.FeatureURL}}" style='{{- template "style_text_feature_link" -}}'>{{.FeatureName}}</a>
            </div>
        </div>
    {{- end -}}
</div>
{{- end -}}`

const buttonComponent = `{{- define "button" -}}
<div style='{{- template "style_button_wrapper" -}}'>
    <a href="{{.URL}}" style='{{- template "style_button_link" -}}'>
        {{.Text}}
    </a>
</div>
{{- end -}}`

const footerComponent = `{{- define "footer" -}}
<div style='{{- template "style_footer_wrapper" -}}'>
    <div style='{{- template "style_footer_hr" -}}'></div>
    <div style='{{- template "style_footer_text_wrapper" -}}'>
        <span style='{{- template "style_text_footer" -}}'>You can </span>
        <a href="{{.UnsubscribeURL}}" style='{{- template "style_text_footer_link" -}}'>unsubscribe</a>
        <span style='{{- template "style_text_footer" -}}'> or change any of your alerts on </span>
        <a href="{{.ManageURL}}" style='{{- template "style_text_footer_link" -}}'>webstatus.dev</a>
    </div>
</div>
{{- end -}}`

const bannerComponents = `{{- define "banner_baseline_widely" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_success" -}}'>
    <div style='{{- template "style_banner_icon_wrapper_28" -}}'>
        <img src="{{.LogoURL}}" height="28" alt="Widely Available" style='{{- template "style_img_responsive" -}}' />
    </div>
    <div style='{{- template "style_banner_text_wrapper" -}}'>
        <span style='{{- template "style_text_banner_bold" -}}'>Baseline</span>
        <span style='{{- template "style_text_banner_normal" -}}'> Widely available </span>
    </div>
</div>
{{- end -}}
{{- define "banner_baseline_newly" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_info" -}}'>
    <div style='{{- template "style_banner_icon_wrapper_28" -}}'>
        <img src="{{.LogoURL}}" height="28" alt="Newly Available" style='{{- template "style_img_responsive" -}}' />
    </div>
    <div style='{{- template "style_banner_text_wrapper" -}}'>
        <span style='{{- template "style_text_banner_bold" -}}'>Baseline</span>
        <span style='{{- template "style_text_banner_normal" -}}'> Newly available </span>
    </div>
</div>
{{- end -}}
{{- define "banner_baseline_regression" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_neutral" -}}'>
    <div style='{{- template "style_banner_icon_wrapper_28" -}}'>
        <img src="{{.LogoURL}}" height="28" alt="Regressed" style='{{- template "style_img_responsive" -}}' />
    </div>
    <div style='{{- template "style_banner_text_wrapper" -}}'>
        <span style='{{- template "style_text_banner_bold" -}}'>Regressed</span>
        <span style='{{- template "style_text_banner_normal" -}}'> to limited availability</span>
    </div>
</div>
{{- end -}}
{{- define "banner_browser_implementation" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_neutral" -}}'>
    <div style='{{- template "style_banner_browser_logos_wrapper" -}}'>
        {{- /* Always display the 4 main browser logos as requested */ -}}
        <div style='{{- template "style_banner_icon_wrapper_20" -}}'>
            <img src="{{browserLogoURL "chrome"}}" height="20" style='{{- template "style_img_responsive" -}}' />
        </div>
        <div style='{{- template "style_banner_icon_wrapper_20" -}}'>
            <img src="{{browserLogoURL "edge"}}" height="20" style='{{- template "style_img_responsive" -}}' />
        </div>
        <div style='{{- template "style_banner_icon_wrapper_20" -}}'>
            <img src="{{browserLogoURL "firefox"}}" height="20" style='{{- template "style_img_responsive" -}}' />
        </div>
        <div style='{{- template "style_banner_icon_wrapper_20" -}}'>
            <img src="{{browserLogoURL "safari"}}" height="20" style='{{- template "style_img_responsive" -}}' />
        </div>
    </div>
    <div style='{{- template "style_banner_text_wrapper" -}}; {{- template "style_text_banner_normal" -}}'>Browser support changed</div>
</div>
{{- end -}}
{{- define "banner_generic" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_neutral" -}}'>
    <div style='{{- template "style_banner_text_wrapper" -}}'>
        <span style='{{- template "style_text_banner_bold" -}}'>{{.Type}}</span>
    </div>
</div>
{{- end -}}`
const featureTitleRowComponent = `{{- define "feature_title_row" -}}
<div style='{{- template "style_feature_title_row_wrapper" -}}'>
    <div style='{{- template "style_feature_title_row_inner" -}}'>
        <a href="{{.URL}}" style='{{- template "style_text_feature_link" -}}'>{{.Name}}</a>
        {{- with .Docs -}}
            {{- if .MDNDocs -}}
            <span style='{{- template "style_text_doc_punctuation" -}}'> (</span>
            {{- range $i, $doc := .MDNDocs }}
                {{- if $i }}, {{ end -}}
                <a href="{{$doc.URL}}" style='{{- template "style_text_doc_link" -}}'>MDN</a>
            {{- end -}}
            <span style='{{- template "style_text_doc_punctuation" -}}'>)</span>
            {{- end -}}
        {{- end -}}
    </div>
    {{- if .Date -}}
    <div style='{{- template "style_text_date" -}}'>{{.Date}}</div>
    {{- end -}}
</div>
{{- end -}}`

const browserStatusDetailComponent = `{{- define "browser_status_detail" -}}
    {{- formatBrowserStatus .Status -}}
    {{- if .Version -}}
        <span style='{{- template "color_text_medium" -}}'> in {{.Version}}</span>
    {{- end -}}
    {{- if .Date -}}
        <span style='{{- template "color_text_medium" -}}'> (on {{ formatDate .Date -}})</span>
    {{- end -}}
{{- end -}}`

const EmailComponents = badgeComponent +
	introTextComponent +
	changeDetailComponent +
	baselineChangeItemComponent +
	browserItemComponent +
	buttonComponent +
	footerComponent +
	bannerComponents +
	featureTitleRowComponent +
	browserStatusDetailComponent
