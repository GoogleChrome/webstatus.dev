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

const styleSnippets = `{{- define "font_family_main" -}}font-family: SF Pro, system-ui, sans-serif;{{- end -}}
{{- define "font_family_monospace" -}}font-family: Menlo, monospace;{{- end -}}
{{- define "font_weight_bold" -}}font-weight: 700;{{- end -}}
{{- define "font_weight_normal" -}}font-weight: 400;{{- end -}}
{{- define "color_text_dark" -}}color: #18181B;{{- end -}}
{{- define "color_text_medium" -}}color: #52525B;{{- end -}}
{{- define "color_bg_success" -}}background: #E6F4EA;{{- end -}}
{{- define "color_bg_info" -}}background: #E8F0FE;{{- end -}}
{{- define "color_bg_neutral" -}}background: #E4E4E7;{{- end -}}
{{- define "color_bg_light_neutral" -}}background: #F4F4F5;{{- end -}}`

const layoutStyles = `{{- define "style_body_wrapper" -}}max-width: 600px; margin: 0 auto; padding: 20px;{{- end -}}
{{- define "style_subject_header" -}}{{- template "style_text_normal" -}}; align-self: stretch; margin: 0;{{- end -}}
{{- define "style_query_text" -}}{{- template "style_text_normal" -}}; align-self: stretch;{{- end -}}
{{- define "style_section_wrapper" -}}align-self: stretch; padding-top: 8px; padding-bottom: 8px; flex-direction: column; justify-content: flex-start; align-items: flex-start; display: flex;{{- end -}}
{{- define "style_card_body" -}}align-self: stretch; padding-top: 12px; padding-bottom: 15px; padding-left: 15px; padding-right: 15px; overflow: hidden; border-bottom-right-radius: 4px; border-bottom-left-radius: 4px; border-left: 1px solid #E4E4E7; border-right: 1px solid #E4E4E7; border-bottom: 1px solid #E4E4E7; flex-direction: column; justify-content: center; align-items: flex-start; display: flex; background: #FFFFFF;{{- end -}}
{{- define "style_button_link" -}}display: inline-block; padding: 10px 20px; background: #18181B; color: white; text-decoration: none; border-radius: 4px; {{- template "font_family_main" -}}; font-weight: 500; font-size: 14px;{{- end -}}`

const composedTextStyles = `{{- define "style_text_badge_title" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_bold" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_badge_description" -}}{{- template "color_text_medium" -}}; font-size: 12px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_normal" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 21px; word-wrap: break-word;{{- end -}}
{{- define "style_text_body" -}}{{- template "color_text_dark" -}}; font-size: 16px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 30.40px; word-wrap: break-word;{{- end -}}
{{- define "style_text_body_subtle" -}}{{- template "color_text_medium" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 26.60px; word-wrap: break-word;{{- end -}}
{{- define "style_text_banner_bold" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_bold" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_banner_normal" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_feature_link" -}}{{- template "color_text_dark" -}}; font-size: 16px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; text-decoration: underline; line-height: 30.40px; word-wrap: break-word;{{- end -}}
{{- define "style_text_doc_link" -}}{{- template "color_text_medium" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; text-decoration: underline; line-height: 26.60px;{{- end -}}
{{- define "style_text_doc_punctuation" -}}{{- template "color_text_medium" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 26.60px;{{- end -}}
{{- define "style_text_date" -}}{{- template "color_text_medium" -}}; font-size: 12px; {{- template "font_family_monospace" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_browser_item" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_footer" -}}{{- template "color_text_medium" -}}; font-size: 11px; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_footer_link" -}}{{- template "color_text_medium" -}}; font-size: 11px; {{- template "font_weight_normal" -}}; text-decoration: underline; word-wrap: break-word;{{- end -}}`

const EmailStyles = styleSnippets + layoutStyles + composedTextStyles + `{{- define "style_body" -}}{{- template "font_family_main" -}}; line-height: 1.5; color: #333; margin: 0; padding: 0;{{- end -}}
{{- define "style_change_detail_div" -}}align-self: stretch; justify-content: flex-start; align-items: center; gap: 10px; display: inline-flex; width: 100%;{{- end -}}
{{- define "style_change_detail_inner_div" -}}flex: 1 1 0;{{- end -}}
{{- define "style_split_into" -}}flex: 1 1 0;{{- end -}}`
