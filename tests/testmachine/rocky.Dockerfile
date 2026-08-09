FROM rockylinux/rockylinux:10

RUN set -eux && \
  dnf install -y \
    openssh-server \
    python3 \
    sudo

RUN sed -i 's/^#\(PermitRootLogin\) .*/\1 yes/' /etc/ssh/sshd_config

RUN useradd -m rocky
RUN echo 'rocky ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/90-cloud-init-users

RUN mkdir /var/run/sshd

RUN printf 'root:hunter2\nrocky:hunter2\n' | chpasswd
RUN ssh-keygen -A

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
