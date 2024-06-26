package alicloud

import (
	"fmt"
	"testing"

	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func init() {
	resource.AddTestSweepers("alicloud_rds_clone_db_instance", &resource.Sweeper{
		Name: "alicloud_rds_clone_db_instance",
		F:    testSweepDBInstances,
	})
}

/*
Because the backup will be automatically distributed when the instance is created,
and manually distributed backup will check whether there are currently running backup tasks,
so the instance created in advance will be used for automatic testing.
*/
func resourceCloneDBInstanceConfigDependence(name string) string {
	return fmt.Sprintf(`
variable "name" {
	default = "%s"
}

data "alicloud_db_zones" "default" {
  engine                   = "PostgreSQL"
  engine_version           = "13.0"
  instance_charge_type     = "PostPaid"
  category                 = "HighAvailability"
  db_instance_storage_type = "cloud_essd"
}

data "alicloud_db_instance_classes" "default" {
  zone_id                  = data.alicloud_db_zones.default.zones.0.id
  engine                   = "PostgreSQL"
  engine_version           = "13.0"
  category                 = "HighAvailability"
  db_instance_storage_type = "cloud_essd"
  instance_charge_type     = "PostPaid"
}

data "alicloud_vpcs" "default" {
    name_regex = "^default-NODELETING$"
}

data "alicloud_vswitches" "default" {
  vpc_id = data.alicloud_vpcs.default.ids.0
  zone_id = data.alicloud_db_zones.default.zones.0.id
}

resource "alicloud_vswitch" "this" {
 count = length(data.alicloud_vswitches.default.ids) > 0 ? 0 : 1
 vswitch_name = var.name
 vpc_id = data.alicloud_vpcs.default.ids.0
 zone_id = data.alicloud_db_zones.default.ids.0
 cidr_block = cidrsubnet(data.alicloud_vpcs.default.vpcs.0.cidr_block, 8, 4)
}
locals {
  vswitch_id = length(data.alicloud_vswitches.default.ids) > 0 ? data.alicloud_vswitches.default.ids.0 : concat(alicloud_vswitch.this.*.id, [""])[0]
  zone_id = data.alicloud_db_zones.default.ids[length(data.alicloud_db_zones.default.ids)-1]
}

data "alicloud_db_instances" "default" {
  name_regex = "^default-PostgreSQL-NODELETING$"
}

resource "alicloud_rds_backup" "default" {
  db_instance_id    = data.alicloud_db_instances.default.instances.0.id
  remove_from_state = "true"
}
`, name)
}

var cloneInstanceBasicMap = map[string]string{}

func TestAccAlicloudRdsCloneDBInstancePostgreSQLSSL(t *testing.T) {
	var instance map[string]interface{}
	var ips []map[string]interface{}
	resourceId := "alicloud_rds_clone_db_instance.default"
	ra := resourceAttrInit(resourceId, cloneInstanceBasicMap)
	rc := resourceCheckInitWithDescribeMethod(resourceId, &instance, func() interface{} {
		return &RdsService{testAccProvider.Meta().(*connectivity.AliyunClient)}
	}, "DescribeDBInstance")
	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	name := "tf-testAccDBInstanceConfig"
	testAccConfig := resourceTestAccConfigFunc(resourceId, name, resourceCloneDBInstanceConfigDependence)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		CheckDestroy:  nil,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(map[string]interface{}{
					"source_db_instance_id":    "${data.alicloud_db_instances.default.instances.0.id}",
					"db_instance_storage_type": "cloud_essd",
					"payment_type":             "PayAsYouGo",
					"backup_id":                "${alicloud_rds_backup.default.backup_id}",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"source_db_instance_id":    CHECKSET,
						"db_instance_storage_type": "cloud_essd",
						"payment_type":             "PayAsYouGo",
						"backup_id":                CHECKSET,
						"engine_version":           "13.0",
						"db_instance_class":        CHECKSET,
						"db_instance_storage":      CHECKSET,
						"zone_id":                  CHECKSET,
						"connection_string":        CHECKSET,
						"port":                     CHECKSET,
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"db_instance_description": "tf-testAccDBInstance_instance_name",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"db_instance_description": "tf-testAccDBInstance_instance_name",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"tcp_connection_type": "SHORT",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"tcp_connection_type": "SHORT",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"deletion_protection": "true",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"deletion_protection": "true",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"db_instance_class": "${data.alicloud_db_instance_classes.default.instance_classes.1.instance_class}",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"db_instance_class": CHECKSET,
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"security_ips": []string{"10.168.1.12", "100.69.7.112"},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKeyValueInMaps(ips, "security ip", "security_ips", "10.168.1.12,100.69.7.112"),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"port":                     "3333",
					"connection_string_prefix": "rm-ccccccc",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"port":                     "3333",
						"connection_string_prefix": "rm-ccccccc",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"ssl_enabled": "1",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"ssl_enabled":     "1",
						"ca_type":         "aliyun",
						"acl":             "perfer",
						"replication_acl": "perfer",
						"server_cert":     CHECKSET,
						"server_key":      CHECKSET,
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"ssl_enabled":                 "1",
					"ca_type":                     "aliyun",
					"client_ca_enabled":           "1",
					"client_ca_cert":              client_ca_cert,
					"client_crl_enabled":          "1",
					"client_cert_revocation_list": client_cert_revocation_list,
					"acl":                         "cert",
					"replication_acl":             "cert",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"ssl_enabled":                 "1",
						"ca_type":                     "aliyun",
						"client_ca_enabled":           "1",
						"client_ca_cert":              client_ca_cert2,
						"client_crl_enabled":          "1",
						"client_cert_revocation_list": client_cert_revocation_list2,
						"acl":                         "cert",
						"replication_acl":             "cert",
						"server_cert":                 CHECKSET,
						"server_key":                  CHECKSET,
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"db_instance_description":     "tf-testAccDBInstance_instance_name",
					"security_ips":                []string{"10.168.1.12", "100.69.7.112"},
					"port":                        "3333",
					"connection_string_prefix":    "rm-ccccccc",
					"ssl_enabled":                 "1",
					"ca_type":                     "aliyun",
					"client_ca_enabled":           "1",
					"client_ca_cert":              client_ca_cert,
					"client_crl_enabled":          "1",
					"client_cert_revocation_list": client_cert_revocation_list,
					"acl":                         "cert",
					"replication_acl":             "cert",
					"deletion_protection":         "false",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"ssl_enabled":                 "1",
						"ca_type":                     "aliyun",
						"client_ca_enabled":           "1",
						"client_ca_cert":              client_ca_cert2,
						"client_crl_enabled":          "1",
						"client_cert_revocation_list": client_cert_revocation_list2,
						"acl":                         "cert",
						"replication_acl":             "cert",
						"server_cert":                 CHECKSET,
						"server_key":                  CHECKSET,
						"deletion_protection":         "false",
					}),
				),
			},
			{
				ResourceName:            resourceId,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_restart", "backup_id", "source_db_instance_id", "client_ca_enabled", "client_crl_enabled", "connection_string_prefix", "ssl_enabled", "pg_hba_conf"},
			},
		},
	})
}

// SSL function and pg_hba_conf function incompatible, so add this test case for pg_hba_conf without ssl function.
func TestAccAlicloudRdsCloneDBInstancePostgreSQL_PG_HBA_CONF(t *testing.T) {
	var instance map[string]interface{}
	var ips []map[string]interface{}
	resourceId := "alicloud_rds_clone_db_instance.default"
	ra := resourceAttrInit(resourceId, cloneInstanceBasicMap)
	rc := resourceCheckInitWithDescribeMethod(resourceId, &instance, func() interface{} {
		return &RdsService{testAccProvider.Meta().(*connectivity.AliyunClient)}
	}, "DescribeDBInstance")
	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	name := "tf-testAccDBInstanceConfig"
	testAccConfig := resourceTestAccConfigFunc(resourceId, name, resourceCloneDBInstanceConfigDependence)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		CheckDestroy:  nil,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(map[string]interface{}{
					"source_db_instance_id":    "${data.alicloud_db_instances.default.instances.0.id}",
					"db_instance_storage_type": "cloud_essd",
					"payment_type":             "PayAsYouGo",
					"backup_id":                "${alicloud_rds_backup.default.backup_id}",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"source_db_instance_id":    CHECKSET,
						"db_instance_storage_type": "cloud_essd",
						"payment_type":             "PayAsYouGo",
						"backup_id":                CHECKSET,
						"engine_version":           "13.0",
						"db_instance_class":        CHECKSET,
						"db_instance_storage":      CHECKSET,
						"zone_id":                  CHECKSET,
						"connection_string":        CHECKSET,
						"port":                     CHECKSET,
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"db_instance_description": "tf-testAccDBInstance_instance_name",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"db_instance_description": "tf-testAccDBInstance_instance_name",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"deletion_protection": "true",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"deletion_protection": "true",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"pg_hba_conf": []interface{}{
						map[string]interface{}{
							"type":        "host",
							"user":        "all",
							"address":     "0.0.0.0/0",
							"database":    "all",
							"method":      "md5",
							"priority_id": "0",
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"pg_hba_conf.#": "1",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"db_instance_class": "${data.alicloud_db_instance_classes.default.instance_classes.1.instance_class}",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"db_instance_class": CHECKSET,
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"security_ips": []string{"10.168.1.12", "100.69.7.112"},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKeyValueInMaps(ips, "security ip", "security_ips", "10.168.1.12,100.69.7.112"),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"port":                     "3333",
					"connection_string_prefix": "rm-ccccccc",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"port":                     "3333",
						"connection_string_prefix": "rm-ccccccc",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"db_instance_description":  "tf-testAccDBInstance_instance_name",
					"security_ips":             []string{"10.168.1.12", "100.69.7.112"},
					"port":                     "3333",
					"connection_string_prefix": "rm-ccccccc",
					"deletion_protection":      "false",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"deletion_protection": "false",
					}),
				),
			},
			{
				ResourceName:            resourceId,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_restart", "backup_id", "source_db_instance_id", "connection_string_prefix", "pg_hba_conf"},
			},
		},
	})
}

func TestAccAlicloudRdsCloneDBInstanceMySQL_Serverless(t *testing.T) {
	var instance map[string]interface{}
	resourceId := "alicloud_rds_clone_db_instance.default"
	ra := resourceAttrInit(resourceId, cloneInstanceBasicMap)
	rc := resourceCheckInitWithDescribeMethod(resourceId, &instance, func() interface{} {
		return &RdsService{testAccProvider.Meta().(*connectivity.AliyunClient)}
	}, "DescribeDBInstance")
	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	name := "tf-testAccDBInstanceConfig-CloneMySQLServerless"
	testAccConfig := resourceTestAccConfigFunc(resourceId, name, resourceCloneDBInstanceConfigDependence_MySQLServerless)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckWithRegions(t, true, connectivity.MySQLServerlessSupportRegions)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		CheckDestroy:  nil,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(map[string]interface{}{
					"source_db_instance_id":    "${data.alicloud_db_instances.default.instances.0.id}",
					"db_instance_class":        "${data.alicloud_db_instances.default.instances.0.instance_type}",
					"zone_id":                  "${data.alicloud_db_instances.default.instances.0.availability_zone}",
					"instance_network_type":    "VPC",
					"vpc_id":                   "${data.alicloud_db_instances.default.instances.0.vpc_id}",
					"vswitch_id":               "${data.alicloud_db_instances.default.instances.0.vswitch_id}",
					"category":                 "serverless_basic",
					"db_instance_storage_type": "cloud_essd",
					"payment_type":             "Serverless",
					"db_instance_storage":      "${data.alicloud_db_instances.default.instances.0.instance_storage}",
					"backup_id":                "${alicloud_rds_backup.default.backup_id}",
					"serverless_config": []interface{}{
						map[string]interface{}{
							"max_capacity": "8",
							"min_capacity": "0.5",
							"auto_pause":   false,
							"switch_force": false,
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"source_db_instance_id":            CHECKSET,
						"db_instance_class":                CHECKSET,
						"zone_id":                          CHECKSET,
						"instance_network_type":            "VPC",
						"vpc_id":                           CHECKSET,
						"vswitch_id":                       CHECKSET,
						"category":                         "serverless_basic",
						"db_instance_storage_type":         "cloud_essd",
						"payment_type":                     "Serverless",
						"db_instance_storage":              CHECKSET,
						"backup_id":                        CHECKSET,
						"serverless_config.#":              "1",
						"serverless_config.0.max_capacity": "8",
						"serverless_config.0.min_capacity": "0.5",
						"serverless_config.0.auto_pause":   "false",
						"serverless_config.0.switch_force": "false",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"serverless_config": []interface{}{
						map[string]interface{}{
							"max_capacity": "7",
							"min_capacity": "1.5",
							"auto_pause":   false,
							"switch_force": false,
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"serverless_config.#":              "1",
						"serverless_config.0.max_capacity": "7",
						"serverless_config.0.min_capacity": "1.5",
						"serverless_config.0.auto_pause":   "false",
						"serverless_config.0.switch_force": "false",
					}),
				),
			},
		},
	})
}

/*
Because the backup will be automatically distributed when the instance is created,
and manually distributed backup will check whether there are currently running backup tasks,
so the instance created in advance will be used for automatic testing.
*/
func resourceCloneDBInstanceConfigDependence_MySQLServerless(name string) string {
	return fmt.Sprintf(`
variable "name" {
	default = "%s"
}

data "alicloud_db_zones" "default"{
    engine = "MySQL"
    engine_version = "8.0"
    instance_charge_type = "Serverless"
    category = "serverless_basic"
    db_instance_storage_type = "cloud_essd"
}

data "alicloud_db_instance_classes" "default" {
    zone_id = data.alicloud_db_zones.default.ids.1
    engine = "MySQL"
    engine_version = "8.0"
    category = "serverless_basic"
    db_instance_storage_type = "cloud_essd"
    instance_charge_type = "Serverless"
    commodity_code = "rds_serverless_public_cn"
}

data "alicloud_vpcs" "default" {
    name_regex = "^default-NODELETING$"
}

data "alicloud_vswitches" "default" {
  vpc_id = data.alicloud_vpcs.default.ids.0
  zone_id = data.alicloud_db_zones.default.ids.1
}

data "alicloud_db_instances" "default" {
  name_regex = "^default-Serverless-NODELETING$"
}

resource "alicloud_rds_backup" "default" {
  db_instance_id    = data.alicloud_db_instances.default.instances.0.id
  remove_from_state = "true"
}

`, name)
}
