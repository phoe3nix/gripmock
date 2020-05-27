FROM golang:alpine

ARG ssh_prv_key
ARG ssh_pub_key
ARG ssh_hosts

RUN mkdir /proto

RUN mkdir /stubs

RUN apk -U --no-cache add git protobuf

RUN apk add openssh-client

# Authorize SSH Host
RUN mkdir -p /root/.ssh && \
    chmod 0700 /root/.ssh

# Add the keys and set permissions
RUN echo "$ssh_prv_key" > /root/.ssh/id_rsa && \
    echo "$ssh_pub_key" > /root/.ssh/id_rsa.pub && \
	echo "$ssh_hosts" > /root/.ssh/known_hosts && \
    chmod 600 /root/.ssh/id_rsa && \
    chmod 600 /root/.ssh/id_rsa.pub && \
	chmod 644 /root/.ssh/known_hosts

RUN go get -u -v github.com/golang/protobuf/protoc-gen-go \
	github.com/mitchellh/mapstructure \
	google.golang.org/grpc \
	google.golang.org/grpc/reflection \
	golang.org/x/net/context \
	github.com/go-chi/chi \
	github.com/lithammer/fuzzysearch/fuzzy \
	golang.org/x/tools/imports

RUN go get -u -v github.com/gobuffalo/packr/v2/... \
                 github.com/gobuffalo/packr/v2/packr2

# cloning well-known-types
RUN git clone https://github.com/google/protobuf.git /protobuf-repo

RUN git clone git@gitlab.cloud.vtblife.ru:vtblife/mobile/common/gripmock.git /master

RUN mkdir protobuf

# only use needed files
RUN mv /protobuf-repo/src/ /protobuf/

RUN rm -rf /protobuf-repo

RUN mkdir -p /go/src/gitlab.cloud.vtblife.ru/vtblife/mobile/common/gripmock

COPY . /go/src/gitlab.cloud.vtblife.ru/vtblife/mobile/common/gripmock

WORKDIR /go/src/gitlab.cloud.vtblife.ru/vtblife/mobile/common/gripmock/protoc-gen-gripmock

RUN cd $GOPATH && packr2

# install generator plugin
RUN go install -v

RUN packr2 clean

WORKDIR /go/src/gitlab.cloud.vtblife.ru/vtblife/mobile/common/gripmock

# install gripmock
RUN go install -v

RUN rm -rf /root/.ssh/

EXPOSE 4770 4771

ENTRYPOINT ["gripmock"]
