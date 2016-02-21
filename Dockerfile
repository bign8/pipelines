FROM scratch
ADD pipeline /
EXPOSE 8000
CMD ["/pipeline"]
