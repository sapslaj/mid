FROM ubuntu:24.04

RUN set -eux && \
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get update && \
  apt-get install -y --no-install-recommends \
    build-essential \
    openssh-server \
    python3 \
    python3-apt \
    sudo

RUN sed -i 's/^#\(PermitRootLogin\) .*/\1 yes/' /etc/ssh/sshd_config
RUN sed -i 's/^\(UsePAM yes\)/# \1/' /etc/ssh/sshd_config

RUN echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/90-cloud-init-users

RUN mkdir /var/run/sshd

RUN printf 'root:hunter2\nubuntu:hunter2\n' | chpasswd

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
