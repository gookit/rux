#
# @build-example docker build . -f Dockerfile -t inherelab/httprr
#

################################################################################
###  builder image
################################################################################
FROM golang:1.14-alpine as Builder

# Recompile the standard library without CGO
#RUN CGO_ENABLED=0 go install -a std

ENV BUILd_DIR=/go/project \
    GO111MODULE=on \
    GOPROXY=https://goproxy.io

RUN mkdir -p $BUILd_DIR

COPY . $BUILd_DIR

# Compile the binary and statically link
# -ldflags '-w -s'
#   -s: 去掉符号表
#   -w: 去掉调试信息，不能gdb调试了
# RUN cd $BUILd_DIR && CGO_ENABLED=0 go build -ldflags '-d -w -s' -o /tmp/app
RUN go version && cd $BUILd_DIR && go build -ldflags '-w -s' -o $BUILd_DIR/app
# RUN cd $BUILd_DIR && go build -o /tmp/app

################################################################################
###  target image
################################################################################
FROM alpine:3.12
LABEL maintainer="inhere <in.798@qq.com>" version="1.0"

##
# ---------- env settings ----------
##
ARG timezone
# env: prod pre test dev. --build-arg app_env=dev
ARG app_env=dev
ARG app_port

ENV APP_ENV=${app_env:-"dev"} \
    APP_PORT=${app_port:-8080} \
    BUILd_DIR=/go/project \
    TIMEZONE=${timezone:-"Asia/Shanghai"}

EXPOSE ${APP_PORT}
WORKDIR /data/www

COPY --from=Builder $BUILd_DIR/* ./
#COPY --from=Builder $BUILd_DIR/static static
#COPY --from=Builder $BUILd_DIR/resource resource
#COPY --from=Builder $BUILd_DIR/app app

##
# ---------- some config, clear work ----------
##
RUN set -ex; \
    sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/' /etc/apk/repositories; \
    # install some tools
    apk update && apk add --no-cache tzdata ca-certificates; \
    # clear caches
    rm -rf /var/cache/apk/*; \
    # - config timezone
    ln -sf /usr/share/zoneinfo/${TIMEZONE} /etc/localtime; \
    echo "${TIMEZONE}" > /etc/timezone; \
    # - create logs, caches dir
    mkdir -p /data/logs /data/www; \
#    chown -R worker:worker /data/www; \
    chown -R www:www /data/www; \
    chmod a+x /data/www/app; \
    # && chown -R www:www /data/logs \
    echo -e "\033[42;37m Build Completed :).\033[0m\n"

CMD ./app
