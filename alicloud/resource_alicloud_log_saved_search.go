package alicloud

import (
	"fmt"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAlicloudLogSavedSearch() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudLogSavedSearchCreate,
		Read:   resourceAlicloudLogSavedSearchRead,
		Update: resourceAlicloudLogSavedSearchUpdate,
		Delete: resourceAlicloudLogSavedSearchDelete,

		Schema: map[string]*schema.Schema{
			"project_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"log_store_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"topic": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"query": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAlicloudLogSavedSearchCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	logService := LogService{client}
	var requestInfo *sls.Client

	savedSearch := sls.SavedSearch{
		SavedSearchName: d.Get("name").(string),
		DisplayName:     d.Get("display_name").(string),
		SearchQuery:     d.Get("query").(string),
		Logstore:        d.Get("log_store_name").(string),
	}

	if v, ok := d.GetOk("topic"); ok {
		savedSearch.Topic = v.(string)
	}

	if err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		_, err := client.WithLogClient(func(slsClient *sls.Client) (interface{}, error) {
			return nil, slsClient.CreateSavedSearch(d.Get("project_name").(string), &savedSearch)
		})
		if err != nil {
			if IsExpectedErrors(err, []string{LogClientTimeout}) {
				time.Sleep(5 * time.Second)
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		addDebug("CreateSavedSearch", savedSearch, requestInfo, map[string]interface{}{
			"savedSearch": savedSearch,
		})
		// 此处的 WaitForSavedSearch 用于确保创建成功， PUBLSHED 在此处为占位符，实际上只要不是 DELETED 即可
		if err := logService.WaitForSavedSearch(d.Get("project_name").(string), d.Get("name").(string), PUBLISHED, DefaultTimeout); err != nil {
			return resource.NonRetryableError(err)
		}
		d.SetId(fmt.Sprintf("%s%s%s", d.Get("project_name").(string), COLON_SEPARATED, d.Get("name").(string)))
		return nil
	}); err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "alicloud_log_savedsearch", "CreateSavedSearch", AliyunLogGoSdkERROR)
	}
	return resourceAlicloudLogSavedSearchRead(d, meta)
}

func resourceAlicloudLogSavedSearchRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	logService := LogService{client}
	parts, err := ParseResourceId(d.Id(), 2)
	if err != nil {
		return WrapError(err)
	}
	object, err := logService.DescribeSavedSearch(parts[0], parts[1])
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return WrapError(err)
	}

	d.Set("project_name", parts[0])
	d.Set("name", object.SavedSearchName)
	d.Set("display_name", object.DisplayName)
	d.Set("log_store_name", object.Logstore)
	d.Set("query", object.SearchQuery)
	d.Set("topic", object.Topic)

	return nil
}

// here
func resourceAlicloudLogSavedSearchUpdate(d *schema.ResourceData, meta interface{}) error {
	parts, err := ParseResourceId(d.Id(), 2)
	if err != nil {
		return WrapError(err)
	}

	update := false
	if d.HasChange("display_name") {
		update = true
	}
	if d.HasChange("topic") {
		update = true
	}
	if d.HasChange("query") {
		update = true
	}

	if update {
		client := meta.(*connectivity.AliyunClient)
		savedSearch := sls.SavedSearch{
			SavedSearchName: d.Get("name").(string),
			DisplayName:     d.Get("display_name").(string),
			SearchQuery:     d.Get("query").(string),
			Logstore:        d.Get("log_store_name").(string),
		}

		if v, ok := d.GetOk("topic"); ok {
			savedSearch.Topic = v.(string)
		}

		_, err = client.WithLogClient(func(slsClient *sls.Client) (interface{}, error) {
			return nil, slsClient.UpdateSavedSearch(parts[0], &savedSearch)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "UpdateSavedSearch", AliyunLogGoSdkERROR)
		}
	}
	return resourceAlicloudLogSavedSearchRead(d, meta)
}

func resourceAlicloudLogSavedSearchDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	logService := LogService{client}
	parts, err := ParseResourceId(d.Id(), 2)
	if err != nil {
		return WrapError(err)
	}
	var requestInfo *sls.Client
	err = resource.Retry(3*time.Minute, func() *resource.RetryError {
		raw, err := client.WithLogClient(func(slsClient *sls.Client) (interface{}, error) {
			requestInfo = slsClient
			return nil, slsClient.DeleteSavedSearch(parts[0], parts[1])
		})
		if err != nil {
			if IsExpectedErrors(err, []string{LogClientTimeout, "RequestTimeout"}) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		addDebug("DeleteSavedSearch", raw, requestInfo, map[string]interface{}{
			"project_name":      parts[0],
			"saved_search_name": parts[1],
		})
		return nil
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"SavedSearchNotExist", "ProjectNotExist"}) {
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteSavedSearch", AliyunLogGoSdkERROR)
	}
	return WrapError(logService.WaitForSavedSearch(parts[0], parts[1], Deleted, DefaultTimeout))
}
