.PHONY: clean
all: data/plain_chroot data/super_chroot

data/plain_chroot: plain_chroot.c
	gcc -static -o data/plain_chroot plain_chroot.c
data/super_chroot: super_chroot.c
	gcc -static -o data/super_chroot super_chroot.c
clean:
	rm -rf data/plain_chroot data/super_chroot