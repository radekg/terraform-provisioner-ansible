FROM hashicorp/terraform:light
RUN apk update && apk add ansible bash
ADD https://github.com/radekg/terraform-provisioner-ansible/releases/download/v2.0.1/terraform-provisioner-ansible-linux-amd64_v2.0.1 /root/.terraform.d/plugins/terraform-provisioner-ansible
RUN chmod 755 /root/.terraform.d/plugins/terraform-provisioner-ansible
