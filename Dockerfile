FROM remotelyplatform/ascode:latest

LABEL MAINTAINER="Remotely Works <platform@remotely.works>"
LABEL "com.github.actions.description"="converts starlark files to HCL"
LABEL "com.github.actions.name"="ascode-action"
LABEL "com.github.actions.color"="blue"

COPY entrypoint.sh /
ENTRYPOINT ["/entrypoint.sh"]
