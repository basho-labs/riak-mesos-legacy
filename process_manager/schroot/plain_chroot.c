#define _GNU_SOURCE
#include <sched.h>
#include <stdio.h>
#include <unistd.h>
#include <signal.h>
#include <stdlib.h>
#include <fcntl.h>
#include <stdlib.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <stdio.h>
#define chk(X) if((X) == -1) { perror(#X); exit(1); }
#include <stdio.h>
#include <sys/types.h>
#include <dirent.h>

int checked_mount(const char *source, const char *target) {
  struct stat mount_data;
  if (stat(target, &mount_data) == -1) {
    printf("Stat error on dir: %s\n", target);
    perror(0);
  }
  if (!S_ISDIR(mount_data.st_mode)) {
    printf("Error, %s is not directory (or found)\n", target);
    exit(1);
  }
  return mount(source, target, 0, MS_BIND|MS_REC, 0);
}

int main(int argc, char **argv) {

	chk(chdir(argv[1]));
	chk(checked_mount("/dev", "./dev"));
	chk(checked_mount("/proc", "./proc"));
 	chk(checked_mount("/sys", "./sys"));
 // This makes the assumption that /etc is on the same partition as /
  struct stat mount_data;
 	if (stat("./parent_root", &mount_data) != -1) {
 	  if (S_ISDIR(mount_data.st_mode)) {
      mount("/", "./parent_root", 0, MS_BIND|MS_REC, 0);
    }
 	}

	chk(chroot("."));
	chk(execvp(argv[2], argv+2));
}