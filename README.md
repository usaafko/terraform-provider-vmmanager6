Terraform provider for ISPsystem VMmanager 6 virtualization platform

## Example config

```
terraform {
  required_providers {
    vmmanager6 = {
      source = "usaafko/vmmanager6"
      version = "> 0.0.5"
    }
  }
}

provider "vmmanager6" {
  pm_email = "admin@example.com"
  pm_password = "___PASSWORD__"
  pm_api_url     = "https://__DOMAIN__"
  # pm_debug = "true"
  # pm_log_enable = "true"
  # pm_log_file = "vm6.log"
  # pm_log_levels = {
  #      _default = "debug"
  # }
}

resource "vmmanager6_network" "net1" {
  network = "1.1.3.0/24"
  gateway = "1.1.3.1"
  desc = "Terraform network"
}

resource "vmmanager6_pool" "pool1" {
  pool = "terraform"
  desc = "Terraform pool"
  ranges = ["1.1.3.4", "1.1.3.10-1.1.3.20"]
  depends_on = [vmmanager6_network.net1]
}

resource "vmmanager6_vm_qemu" "vm2" {
  name = "mein"
  desc = "testing terraform"
  cores = 1
  memory = 1024
  disk = 6000
  os = 1
  password = "@1231sdas"
  cluster = 1
  account = "${vmmanager6_account.user.id}"
  domain = "mein.example.com"
  depends_on = [vmmanager6_network.net1, vmmanager6_pool.pool1, vmmanager6_account.ilya ]
  ipv4_pools = [ "${vmmanager6_pool.pool1.id}" ]
  ipv4_number = 1
  recipes {
        recipe = 12
        recipe_params {
                name = "ZABBIX_SERVER"
                value = "${vmmanager6_vm_qemu.zabbix_server.ip_addresses[0].addr}"
        }
  }
}

resource "vmmanager6_account" "user" {
  email = "user@user.com"
  password = "asdh49@Aas"
  role = "@advanced_user"
}
```
