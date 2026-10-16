# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0


integration {
  name = "ZSphere"
  description = "The ZSphere Cloud plugin can be used with HashiCorp Packer to create custom images on ZSphere Cloud."
  identifier = "packer/ZSvirt/zsvirty"
  docs {
    # Render the generated pages on developer.hashicorp.com.
    process_docs = true
    readme_location = "./README.md"
  }
  license {
    type = "MPL-2.0"
    url = "https://github.com/ZSvirt/packer-plugin-zsvirty/blob/main/LICENSE"
  }
  component {
    type = "builder"
    name = "ZSphere Snapshot"
    slug = "snapshot"
  }
  component {
    type = "builder"
    name = "ZSphere ISO"
    slug = "iso"
  }
  component {
    type = "builder"
    name = "ZSphere"
    slug = "zsvirt"
  }
}
