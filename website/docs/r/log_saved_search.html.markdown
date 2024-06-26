---
subcategory: "Log Service (SLS)"
layout: "alicloud"
page_title: "Alicloud: alicloud_log_saved_search"
sidebar_current: "docs-alicloud-resource-log-saved-search"
description: |-
  Provides a Alicloud Log Saved Search resource.
---

# alicloud\_log\_saved\_search
Simple Log Service provides the saved search feature to save the required data query and analysis operations. You can use a saved search to quickly perform query and analysis operations.
[Refer to details](https://www.alibabacloud.com/help/en/sls/user-guide/saved-search).

-> **NOTE:** Available in 1.201.9 (ShinnyTech forked version only)

## Example Usage

Basic Usage

```terraform
resource "alicloud_log_project" "default" {
  name        = "tf-project"
  description = "tf unit test"
}
resource "alicloud_log_store" "default" {
  project          = "tf-project"
  name             = "tf-logstore"
  retention_period = "3000"
  shard_count      = 1
}
resource "alicloud_log_saved_search" "example" {
  project_name   = "tf-project"
  log_store_name = "tf-logstore"
  display_name   = "tf-saved-search"
  query          = "* | select * from log"
}
```


## Argument Reference

The following arguments are supported:

* `project_name` - (Required, ForceNew) The name of the log project. It is the only in one Alicloud account.
* `name` - (Required, ForceNew) The name of the saved search. It is unique in the project.
* `log_store_name` - (Required, ForceNew) The name of the logstore.
* `display_name` - (Required) The display name of the saved search.
* `topic` - (Optional) The topic of the saved search.
* `query` - (Required) The query statement of the saved search.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the SavedSearch.
