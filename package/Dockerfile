FROM alpine:3.8 as base
RUN apk add -U ca-certificates
ADD build/out/data.tar.gz /image
RUN mkdir -p /image/etc/ssl/certs /image/run /image/var/run /image/tmp /image/lib/modules /image/lib/firmware && \
    cp /etc/ssl/certs/ca-certificates.crt /image/etc/ssl/certs/ca-certificates.crt
RUN cd image/bin && \
    rm -f k2s && \
    ln -s k2s-server k2s

FROM scratch
COPY --from=base /image /
RUN chmod 1777 /tmp
VOLUME /var/lib/k2s 
VOLUME /var/lib/cni
VOLUME /var/log
ENTRYPOINT ["/bin/k2s"]
CMD ["agent"]
