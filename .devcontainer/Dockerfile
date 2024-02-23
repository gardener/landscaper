FROM mcr.microsoft.com/devcontainers/go:1.22

# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y

RUN go install github.com/gardener/landscapercli/landscaper-cli@latest
RUN curl -s https://ocm.software/install.sh | sudo bash