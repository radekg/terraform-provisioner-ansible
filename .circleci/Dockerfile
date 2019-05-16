FROM golang:1.12.5

ARG ANSIBLE_VERSION=2.6.5

# make Apt non-interactive
RUN echo 'APT::Get::Assume-Yes "true";' > /etc/apt/apt.conf.d/90circleci \
    && echo 'DPkg::Options "--force-confnew";' >> /etc/apt/apt.conf.d/90circleci

ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=C.UTF-8

# Set timezone to UTC by default
RUN ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime

RUN apt update \
    && apt install -y python-pip locales \
    && pip install ansible==${ANSIBLE_VERSION} \
    && go get -u golang.org/x/lint/golint \
    && go get -u github.com/Masterminds/glide \
    && locale-gen C.UTF-8 || true

CMD ["/bin/sh", "-c", "make lint && make test-verbose"]
