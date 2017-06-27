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
    #pip install superlance && \
    useradd -s /sbin/nologin sinkit && \
    cd /opt && wget ${GODIST} && tar -xvf *.tar.gz && rm -rf *.tar.gz

RUN mkdir /tmp/protoc-temp && \
    cd /tmp/protoc-temp && \
    wget -q https://github.com/google/protobuf/releases/download/v3.0.0/protoc-3.0.0-linux-x86_64.zip && \
    unzip protoc-3.0.0-linux-x86_64.zip && \
    mv ./bin/protoc ${GOROOT}/bin/ && \
    chmod a+rx ${GOROOT}/bin/protoc && \
    cd / && \
    rm -rf /tmp/protoc-temp

# Unbound
ADD unbound.conf /etc/unbound/unbound.conf

# Supervisor
ADD supervisord.conf /etc/supervisor/conf.d/supervisord.conf
ADD start.sh /usr/bin/start.sh

USER sinkit

RUN mkdir -p ${GOPATH}/src/${GODNSREPO}

#debug
#RUN go get -u github.com/derekparker/delve/cmd/dlv && ls -lah ${GOPATH}/bin && cp ${GOPATH}/bin/dlv /home/sinkit/

# GoDNS
ADD *.proto ${GOPATH}/src/${GODNSREPO}/
ADD *.go ${GOPATH}/src/${GODNSREPO}/

RUN cd ${GOPATH}/src/${GODNSREPO}/ && \
    go get -u github.com/golang/protobuf/protoc-gen-go && \
    protoc -I=${GOPATH}/src/${GODNSREPO} --plugin=${GOPATH}/bin/protoc-gen-go --go_out=${GOPATH}/src/${GODNSREPO} ${GOPATH}/src/${GODNSREPO}/sinkit-cache.proto && \
    ls -lah ./ && \
    cd ${GOPATH}/src/${GODNSREPO}/ && \
    go get . && \
    go build && \
    cp godns /home/sinkit/ && \
    rm -rf ${GOPATH}
    # ls -lah ./ && \
    # cd /home/sinkit/ && \
    # ls -lah ./ && \
    # rm -rf ${GOPATH}

USER root
RUN setcap 'cap_net_bind_service=+ep' /home/sinkit/godns

EXPOSE 53/tcp
EXPOSE 53/udp
#EXPOSE 2345

# CMD ["/home/sinkit/dlv", "debug", "github.com/Karm/godns", "--headless", "--listen=:2345", "--log"]

CMD ["/usr/bin/start.sh"]

#gcvis
#RUN cd ${GOPATH}/src && go get -u github.com/davecheney/gcvis && cp ${GOPATH}/bin/gcvis /home/sinkit/ && \
#    sed -i 's~command=/home/sinkit/godns~command=/home/sinkit/gcvis -o=false -p 2345 -i "0.0.0.0" /home/sinkit/godns~' /etc/supervisor/conf.d/supervisord.conf
