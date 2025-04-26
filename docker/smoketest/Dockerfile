FROM ubuntu:24.04

RUN set -eux && \
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get update && \
  apt-get install -y --no-install-recommends \
    build-essential \
    openssh-server \
    python3 \
    python3-apt \
  && \
  sed -i 's/^#\(PermitRootLogin\) .*/\1 yes/' /etc/ssh/sshd_config && \
  sed -i 's/^\(UsePAM yes\)/# \1/' /etc/ssh/sshd_config && \
  mkdir /var/run/sshd

RUN echo 'root:hunter2' | chpasswd

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
