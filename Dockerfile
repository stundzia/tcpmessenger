FROM golang

ADD . /go/src/f.oxy.works/paulius.stundzia/tcpmessenger

RUN go install f.oxy.works/paulius.stundzia/tcpmessenger

ENTRYPOINT /go/bin/tcpmessenger

EXPOSE 8033
EXPOSE 8044