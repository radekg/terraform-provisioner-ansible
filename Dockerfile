FROM golang:1.11.4-alpine
WORKDIR /go/src/github.com/radekg/terraform-provisioner-ansible/
ADD . .
RUN go build -o /go/bin/terraform-provisioner-ansible .

FROM hashicorp/terraform:light
RUN apk update && apk add ansible && mkdir -p /root/.terraform.d/plugins/
COPY --from=0 /go/bin/terraform-provisioner-ansible /root/.terraform.d/plugins/
