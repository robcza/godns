FROM fedora:24
MAINTAINER Michal Karm Babacek <karm@email.cz>
ENV DEPS        unbound libevent python-pip wget unzip git wget sed gawk bc tar procps
ENV GOPATH      /home/sinkit/go
ENV GOROOT      /opt/go
ENV GODNSREPO   github.com/Karm/godns
ENV PATH        ${GOROOT}/bin/:${PATH}
ENV GODIST      https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz

RUN dnf -y update && dnf -y install ${DEPS} && dnf clean all && \
    pip install supervisor && \
    pip install superlance && \
    useradd -s /sbin/nologin sinkit && \
    cd /opt && wget ${GODIST} && tar -xvf *.tar.gz && rm -rf *.tar.gz

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
