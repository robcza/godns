FROM fedora:24
MAINTAINER Michal Karm Babacek <karm@email.cz>
ENV DEPS        unbound libevent go python-pip wget unzip git wget sed gawk bc procps
ENV GOPATH      /home/sinkit/go
ENV GODNSREPO   github.com/Karm/godns

RUN dnf -y update && dnf -y install ${DEPS} && dnf clean all && \
    pip install supervisor && \
    pip install superlance

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

USER root

RUN ls -lah /home/sinkit/
RUN setcap 'cap_net_bind_service=+ep' /home/sinkit/godns

# Unbound
ADD unbound.conf /etc/unbound/unbound.conf

# Supervisor
ADD supervisord.conf /etc/supervisor/conf.d/supervisord.conf

ADD start.sh /usr/bin/start.sh

EXPOSE 53/tcp
EXPOSE 53/udp

CMD ["/usr/bin/start.sh"]
