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

resource "vmmanager6_account" "user" {
  email = "user@user.com"
  password = "asdh49@Aas"
  role = "@advanced_user"
  ssh_keys {
    name = "testing"
    ssh_pub_key = "ssh-rsa blabla"
  }
}

resource "vmmanager6_vxlan" "vxlan1" {
  name = "test"
  account = "${vmmanager6_account.user.id}"
  depends_on = [ vmmanager6_account.user ]
  clusters = [ 1 ]
  comment = "Testing VxLAN from Terraform"
  ipnets {
    name = "10.0.0.0/24"
    gateway = "10.0.0.1"
  }
}

resource "vmmanager6_vm_qemu" "vm1" {
  name = "mein"
  desc = "testing terraform"
  cores = 1
  memory = 1024
  disk = 6000
  os = 1
  password = "@1231sdas"
  cluster = 1
  node        = 1
  anti_spoofing = false
  account = "${vmmanager6_account.user.id}"
  domain = "mein.example.com"
  depends_on = [vmmanager6_network.net1, vmmanager6_pool.pool1, vmmanager6_account.user ]
  ipv4_pools = [ "${vmmanager6_pool.pool1.id}" ]
  ipv4_number = 1
  recipes {
        recipe = 11
  }
  vxlan {
    id = "${vmmanager6_vxlan.vxlan1.id}"
    ipnet = "${vmmanager6_vxlan.vxlan1.ipnets[0].id}"
  }
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
  depends_on = [vmmanager6_account.user, vmmanager6_vm_qemu.vm1, vmmanager6_vxlan.vxlan1 ]
  vxlan {
    id = "${vmmanager6_vxlan.vxlan1.id}"
    ipnet = "${vmmanager6_vxlan.vxlan1.ipnets[0].id}"
  }
  recipes {
        recipe = 12
        recipe_params {
                name = "SERVER"
                value = "${vmmanager6_vm_qemu.vm1.ip_addresses[1].addr}"
        }
        recipe_params {
                name = "COMMENT"
                value = "Some comment"
        }
  }
}
```
