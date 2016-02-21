FROM scratch
MAINTAINER Nate Woods <me@bign8.info>
ADD pipeline /
EXPOSE 4222
ENTRYPOINT ["/pipeline"]
CMD ["-h"]
