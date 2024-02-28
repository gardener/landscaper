FROM mcr.microsoft.com/devcontainers/go:1.22

# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y

  # add cert authority for ssl
RUN mkdir -p /registry/certs && \
openssl req -newkey rsa:4096 -nodes -sha256 -keyout /registry/certs/ociregistry.key -addext "subjectAltName = DNS:localhost" -x509 -days 365 -out /registry/certs/ociregistry.crt -subj "/C=DE/ST=Baden-Wuertemberg/L=Walldorf/O=SAP-LOCAL-DEV/OU=ocm/CN=localhost"
RUN  cp /registry/certs/ociregistry.crt /usr/local/share/ca-certificates && \
  update-ca-certificates


RUN go install github.com/gardener/landscapercli/landscaper-cli@latest
RUN curl -s https://ocm.software/install.sh | sudo bash