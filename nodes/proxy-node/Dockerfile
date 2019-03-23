FROM golang:alpine AS build-env
WORKDIR /root/sentinel
COPY dante.go /root/sentinel/
RUN apk update \
    && apk add ca-certificates git \
    && go get github.com/fatih/color github.com/jasonlvhit/gocron \
    && cd /root/sentinel \
    && go build -o dante dante.go


FROM sentinelofficial/stt1-dvpn-openvpn

COPY alpine-dante dante /usr/bin/

COPY app.py /root/app.py
COPY run.sh /root/run.sh
COPY --from=build-env /root/sentinel/dante /root/sentinel/dante

ENV PASS "defaultPasswordFortheNode"

RUN printf 'defaultPasswordFortheNode\ndefaultPasswordFortheNode\n' | adduser sentinel

ADD sockd.conf /etc/sockd.conf

RUN set -x \
     && apk add --no-cache \
         linux-pam \
     && apk add --no-cache -t .build-deps \
             build-base \
             curl \
             linux-pam-dev \
     && cd /tmp \
     && curl -L https://www.inet.no/dante/files/dante-1.4.2.tar.gz | tar xz \
     && cd dante-* \
     && ac_cv_func_sched_setscheduler=no ./configure \
     && make install \
     && cd / \
     && adduser -S -D -u 8062 -H sockd \
     && curl -Lo /usr/local/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.1.3/dumb-init_1.1.3_amd64 \
     && chmod +x /usr/local/bin/dumb-init \
     && rm -rf /tmp/*
    #  && apk del --purge .

EXPOSE 1080 3030

