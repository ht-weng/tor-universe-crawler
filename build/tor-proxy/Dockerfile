FROM alpine:latest
LABEL maintainer="Rui NI <ranqus@gmail.com>"

ENV TOR_SOURCE_GIT_REPOSITORY=https://git.torproject.org/tor.git \
    TOR_SOURCE_GIT_BRANCH=tor-0.4.3.6 \
    OBFS4_SOURCE_GIT_REPOSITORY=https://git.torproject.org/pluggable-transports/obfs4.git \
    OBFS4_SOURCE_GIT_BRANCH=obfs4proxy-0.0.11 \
    MEEK_SOURCE_GIT_REPOSITORY=https://git.torproject.org/pluggable-transports/meek.git \
    MEEK_SOURCE_GIT_BRANCH=v0.35.0 \
    TOR_CUSTOM_CONFIGURATION=

RUN set -ex && \
    cd / && \
    echo '#!/bin/sh'$'\n\n'"res=0; for i in \$(seq 0 29); do \$@; res=\$?; [ \$res == 0 ] && exit \$res || sleep 10; done; exit \$res" > /try.sh && chmod +x /try.sh && \
    echo '#!/bin/sh'$'\n\n'"cpid=\"\"; ret=0; i=0; for c in \"\$@\"; do ( (((((eval \$c; echo \$? >&3) | sed \"s/^/|-(\$i) /\" >&4) 2>&1 | sed \"s/^/|-(\$i)!/\" >&2) 3>&1) | (read xs; exit \$xs))  4>&1) & ppid=\$!; cpid=\"\$cpid \$ppid\"; echo \"+ Child \$i (PID \$ppid): \$c ...\"; i=\$((i+1)); done; for c in \$cpid; do wait \$c; cret=\$?; [ \$cret -eq 0 ] && continue; echo \"* Child PID \$c has failed.\" >&2; ret=\$cret; done; exit \$ret" > /child.sh && chmod +x /child.sh && \
    /try.sh apk add --no-cache --virtual .build-deps \
        bash \
        autoconf \
        automake \
        build-base \
        musl-dev \
        libevent-dev \
        openssl-dev \
        zlib-dev \
        git \
        go && \
    mkdir /tmp/.build -p && \
    ([ -z $HTTP_PROXY ] || git config --global http.proxy "$HTTP_PROXY") && \
    ([ -z $HTTPS_PROXY ] || git config --global https.proxy "$HTTPS_PROXY") && \
    /child.sh \
        "/try.sh git clone --depth 1 --branch \"\$TOR_SOURCE_GIT_BRANCH\" \"\$TOR_SOURCE_GIT_REPOSITORY\" /tmp/.build/tor" \
        "/try.sh git clone --depth 1 --branch \"\$OBFS4_SOURCE_GIT_BRANCH\" \"\$OBFS4_SOURCE_GIT_REPOSITORY\" /tmp/.build/obfs4" \
        "/try.sh git clone --depth 1 --branch \"\$MEEK_SOURCE_GIT_BRANCH\" \"\$MEEK_SOURCE_GIT_REPOSITORY\" /tmp/.build/meek" \
    && \
    /child.sh \
        "cd /tmp/.build/tor && /try.sh ./autogen.sh && /try.sh ./configure --disable-asciidoc && /try.sh make && /try.sh make install" \
        "/child.sh \
            \"cd /tmp/.build/obfs4 && /try.sh go build -o /usr/local/bin/obfs4proxy ./obfs4proxy\" \
            \"cd /tmp/.build/meek/meek-client && /try.sh go get -d ./... && /try.sh go build -o /usr/local/bin/meek-client\" \
            \"cd /tmp/.build/meek/meek-server && /try.sh go get -d ./... && /try.sh go build -o /usr/local/bin/meek-server\" \
        " \
    && \
    cd / && \
    apk del .build-deps && \
    /try.sh apk add --no-cache libgcc libevent openssl zlib ca-certificates && \
    rm /try.sh /child.sh /tmp/* /tmp/.[!.]* /root/* /root/.[!.]* /var/cache/apk/* /var/log/* /usr/local/share/tor/* /usr/local/etc/tor/* -rf && \
    adduser -D tor_user && \
    tor --version

ADD torrc /usr/local/etc/tor/
ADD tor.sh /
ADD torrc /

EXPOSE 9050
USER tor_user
ENTRYPOINT [ "/tor.sh" ]
CMD []
