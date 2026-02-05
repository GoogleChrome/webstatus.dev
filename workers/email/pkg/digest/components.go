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
<div style='{{- template "style_intro_wrapper" -}}'>
    <h2 style='{{- template "style_subject_header" -}}'>{{.FullSubject}}</h2>
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
        <table width="100%" border="0" cellspacing="0" cellpadding="0" style="border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt;">
            <tr>
                <td width="36" align="left" valign="middle">
                    <img src="{{.ToURL}}" height="28" alt="{{.To}}" style='{{- template "style_img_responsive" -}}' />
                </td>
                <td align="left" valign="middle">
                    <span style='{{- template "style_text_banner_bold" -}}'>Baseline</span>
                    <span style='{{- template "style_text_banner_normal" -}}'> {{.To}} </span>
                </td>
            </tr>
        </table>
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

const browserItemComponent = `{{- define "browser_change_row" -}}
    <div style='{{- template "style_browser_item_row" -}}'>
        <table border="0" cellspacing="0" cellpadding="0" style="border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt;">
            <tr>
                <td align="left" valign="middle" style="padding-right: 10px;">
                    <img src="{{.LogoURL}}" height="20" alt="{{.Name}}" style='{{- template "style_img_responsive" -}}' />
                </td>
                <td align="left" valign="middle">
                    <div style='{{- template "style_text_browser_item" -}}'>
                        {{.Name}}: {{ template "browser_status_detail" .From }} &rarr; {{ template "browser_status_detail" .To -}}
                    </div>
                </td>
            </tr>
        </table>
    </div>
{{- end -}}

{{- define "browser_item" -}}
<div style='{{- template "style_card_body" -}}'>
    {{- template "browser_change_row" . -}}
    {{- if .FeatureName -}}
        <div style='{{- template "style_browser_item_feature_link_wrapper" -}}'>
            <div style='{{- template "style_banner_text_wrapper" -}}'>
                <a href="{{.FeatureURL}}" style='{{- template "style_text_feature_link" -}}'>{{.FeatureName}}</a>
            </div>
        </div>
    {{- end -}}
    {{- if eq .Type "Removed" -}}
        <div style='{{- template "style_text_warning" -}}'>⚠️ This feature no longer matches your saved search. Please update your saved search if you wish to continue tracking it.</div>
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
    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt;">
        <tr>
            <td width="36" align="left" valign="middle">
                <img src="{{.LogoURL}}" height="28" alt="Widely Available" style='{{- template "style_img_responsive" -}}' />
            </td>
            <td align="left" valign="middle">
                <span style='{{- template "style_text_banner_bold" -}}'>Baseline</span>
                <span style='{{- template "style_text_banner_normal" -}}'> Widely available </span>
            </td>
        </tr>
    </table>
</div>
{{- end -}}
{{- define "banner_baseline_newly" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_info" -}}'>
    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt;">
        <tr>
            <td width="36" align="left" valign="middle">
                <img src="{{.LogoURL}}" height="28" alt="Newly Available" style='{{- template "style_img_responsive" -}}' />
            </td>
            <td align="left" valign="middle">
                <span style='{{- template "style_text_banner_bold" -}}'>Baseline</span>
                <span style='{{- template "style_text_banner_normal" -}}'> Newly available </span>
            </td>
        </tr>
    </table>
</div>
{{- end -}}
{{- define "banner_baseline_regression" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_neutral" -}}'>
    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt;">
        <tr>
            <td width="36" align="left" valign="middle">
                <img src="{{.LogoURL}}" height="28" alt="Regressed" style='{{- template "style_img_responsive" -}}' />
            </td>
            <td align="left" valign="middle">
                <span style='{{- template "style_text_banner_bold" -}}'>Regressed</span>
                <span style='{{- template "style_text_banner_normal" -}}'> to limited availability</span>
            </td>
        </tr>
    </table>
</div>
{{- end -}}
{{- define "banner_browser_implementation" -}}
<div style='{{- template "style_banner_wrapper" -}}{{- template "color_bg_neutral" -}}'>
    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt;">
        <tr>
            <td width="28" align="left" valign="middle">
                 <img src="{{browserLogoURL "chrome"}}" height="20" style='{{- template "style_img_responsive" -}}' />
            </td>
            <td width="28" align="left" valign="middle">
                 <img src="{{browserLogoURL "edge"}}" height="20" style='{{- template "style_img_responsive" -}}' />
            </td>
            <td width="28" align="left" valign="middle">
                 <img src="{{browserLogoURL "firefox"}}" height="20" style='{{- template "style_img_responsive" -}}' />
            </td>
            <td width="28" align="left" valign="middle">
                 <img src="{{browserLogoURL "safari"}}" height="20" style='{{- template "style_img_responsive" -}}' />
            </td>
            <td align="left" valign="middle" style="padding-left: 8px;">
                 <span style='{{- template "style_text_banner_normal" -}}'>Browser support changed</span>
            </td>
        </tr>
    </table>
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
<table width="100%" border="0" cellspacing="0" cellpadding="0" style="border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt;">
    <tr>
        <td align="left" valign="top">
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
        </td>
        {{- if .Date -}}
        <td align="right" valign="top" style="white-space: nowrap; padding-left: 10px;">
            <div style='{{- template "style_text_date" -}}'>{{.Date}}</div>
        </td>
        {{- end -}}
    </tr>
</table>
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
