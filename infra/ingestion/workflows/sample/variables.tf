variable "env_id" {
  type = string
}

variable "regions" {
  type = list(string)
}

variable "sample_custom_step_region_to_step_info_map" {
  type = map(object({
    name = string
    url  = string
  }))
}