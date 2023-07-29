FROM golang:latest as builder
COPY .. /src/asyncjob
RUN go env -w GOPROXY=https://goproxy.cn,direct && cd /src/asyncjob && make all &&  \
    /bin/cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' > /etc/timezone

FROM golang:latest as prod
COPY --from=builder /src/asyncjob /asyncjob
WORKDIR /asyncjob
ENTRYPOINT [ "./bin/asyncjob"]
EXPOSE 8888

