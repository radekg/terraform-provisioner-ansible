---
layout: "docs"
page_title: "Archiving Providers"
sidebar_current: "docs-internals"
description: |-
  Terraform is built on a plugin-based architecture, much of which is maintained by our user community. Occasionally, unmaintained providers may archived to reduce confusion for users and developers.
---

<!--
This page is purposefully not linked from anywhere on terraform.io: it is intended to be linked only from the README files of archived providers.
-->

# Archiving Providers

As contributors' circumstances change, development on a community-maintained Terraform provider can slow. When this happens, HashiCorp may use GitHub's "archiving" feature on the provider's repository, to clearly signal the provider's status to users.

What does archiving mean?

1. The code repository and all commit, issue, and PR history will still be available.
1. Existing released binaries will remain available on the releases site.
1. Documentation for the provider will remain on the Terraform website.
1. Issues and pull requests are not being monitored, merged, or added.
1. No new releases will be published.
1. Nightly acceptance tests may not be run.

HashiCorp may archive a provider when we or the community are not able to support it at a level consistent with our open source guidelines and community expectations.

Archiving is reversible. If anyone from the community is willing to maintain an archived provider, please reach out to the [Terraform Provider Development Program](https://www.terraform.io/guides/terraform-provider-development-program.html) at *terraform-provider-dev@hashicorp.com*.
