FROM fedora:24
MAINTAINER Michal Karm Babacek <karm@email.cz>
ENV DEPS        unbound go supervisor wget unzip git wget
ENV GOPATH      /home/sinkit/go
ENV GODNSREPO   github.com/Karm/godns

RUN dnf -y update && dnf -y install ${DEPS} && dnf clean all

RUN useradd -s /sbin/nologin sinkit

USER sinkit

RUN mkdir -p ${GOPATH}/src/${GODNSREPO}

# GoDNS
ADD *.go ${GOPATH}/src/${GODNSREPO}/
RUN cd ${GOPATH}/src/${GODNSREPO}/ && \
    go get . && \
    go build && \
    cp godns /home/sinkit/ && \
    ls -lah ./ && \
    cd /home/sinkit/ && \
    ls -lah ./ && \
    rm -rf ${GOPATH}
ADD godns.conf /home/sinkit/godns.conf

USER root

RUN ls -lah /home/sinkit/
RUN setcap 'cap_net_bind_service=+ep' /home/sinkit/godns

# Unbound
ADD unbound.conf /etc/unbound/unbound.conf
RUN wget -O /etc/unbound/named.cache ftp://ftp.internic.net./domain/named.cache

# Supervisor
ADD supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 53/tcp
EXPOSE 53/udp

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf", "-n"]
