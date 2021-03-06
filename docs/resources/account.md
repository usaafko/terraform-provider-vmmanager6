---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vmmanager6_account Resource - terraform-provider-vmmanager6"
subcategory: ""
description: |-
  
---

# vmmanager6_account (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `email` (String) User log in to VMmanager by email
- `password` (String, Sensitive) User password

### Optional

- `id` (String) The ID of this resource.
- `role` (String) User role, must be @admin or @advanced_user or @user
- `ssh_keys` (Block List) Set of public ssh keys for account (see [below for nested schema](#nestedblock--ssh_keys))

### Read-Only

- `state` (String) Internal - user state

<a id="nestedblock--ssh_keys"></a>
### Nested Schema for `ssh_keys`

Required:

- `name` (String) name of public ssh key
- `ssh_pub_key` (String) public ssh key

Read-Only:

- `id` (Number) id of public ssh key


