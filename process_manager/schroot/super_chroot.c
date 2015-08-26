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
int main(int argc, char **argv)
{
	uid_t uid = getuid();
	uid_t gid = getgid();

  if (unshare(CLONE_NEWUSER|CLONE_NEWNS) == -1) {
    printf("Error, could not get new namespace, check sysctl kernel.unprivileged_userns_clone = 1\n");
    exit(1);
  }

	char buf[32];

	int fd = open("/proc/self/uid_map", O_RDWR);
	write(fd, buf, snprintf(buf, sizeof buf, "0 %u 1\n", uid));
	close(fd);

	fd = open("/proc/self/setgroups", O_RDWR);
	write(fd, "deny", 4);
	close(fd);

	fd = open("/proc/self/gid_map", O_RDWR);
	write(fd, buf, snprintf(buf, sizeof buf, "0 %u 1\n", gid));
	close(fd);

//	setgroups(0, 0);

	chk(chdir(argv[1]));
	chk(checked_mount("/dev", "./dev"));
	chk(checked_mount("/proc", "./proc"));
 	chk(checked_mount("/sys", "./sys"));

	chk(chroot("."));
	chk(execvp(argv[2], argv+2));
}
