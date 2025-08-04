# ansible

This is a "fork" of Ansible 2.18.5 (commit
[`a3c86e6ac8a321fb25e14ee726e596f3f401549e`](https://github.com/ansible/ansible/commit/a3c86e6ac8a321fb25e14ee726e596f3f401549e)).
It has been modified significantly in order to be used as a library for mid
rather than a standalone tool.

In addition to the builtin collection, portions of the follow additional
collections have been merged in:

- [`ansible.posix`](https://docs.ansible.com/ansible/latest/collections/ansible/posix/index.html)
  (at commit [`dabaca4b70223ea309d8c8af8b9cc9bf48bf1484`](https://github.com/ansible-collections/ansible.posix/commit/dabaca4b70223ea309d8c8af8b9cc9bf48bf1484))
- [`community.general`](https://docs.ansible.com/ansible/latest/collections/community/general/index.html)
  (at commit [`9e317089a8e3566e65b868e107c0df8da88fed30`](https://github.com/ansible-collections/community.general/commit/9e317089a8e3566e65b868e107c0df8da88fed30))
- [`community.docker`](https://docs.ansible.com/ansible/latest/collections/community/docker/index.html)
  (at commit [`da76583d6b96fe225c21927fc523894211c9d939`](https://github.com/ansible-collections/community.docker/commit/da76583d6b96fe225c21927fc523894211c9d939))
