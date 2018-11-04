FROM hashicorp/terraform:light
ARG TAP_VERSION=2.0.1
RUN apk update && apk add ansible bash
ADD https://github.com/radekg/terraform-provisioner-ansible/releases/download/v${TAP_VERSION}/terraform-provisioner-ansible-linux-amd64_v${TAP_VERSION} /root/.terraform.d/plugins/terraform-provisioner-ansible
RUN chmod 755 /root/.terraform.d/plugins/terraform-provisioner-ansible
