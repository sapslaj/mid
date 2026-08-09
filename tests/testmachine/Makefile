.PHONY: default
default: ubuntu-seed.img rocky-seed.img

ubuntu-seed.img: ubuntu-user-data.yaml
	cloud-localds ubuntu-seed.img ubuntu-user-data.yaml

rocky-seed.img: rocky-user-data.yaml
	cloud-localds rocky-seed.img rocky-user-data.yaml
