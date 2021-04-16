#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/stat.h>
#include <unistd.h>
#include <unistd.h>
#include <getopt.h>

#include "common/common.h"
#include <molecule-ipc.h>
#include <chos/errno.h>
#include <global_syscall_protocol.h>
#include <global_syscall_interfaces.h>
#include <molecule_runtime_command_protocol.h>

int start_server() {
	// The file pointer we will associate with the FIFO
	struct sigaction signal_action;
	int fifo_self;
	int fifo_server;
	int global_fifo;
	int ret;
	char* buffer = (char*)malloc(256);
#define MAX_TEST_BUF_SIZE 2048
	char test_buf[MAX_TEST_BUF_SIZE];
	
	register_self_global(GLOBAL_OS_PORT); //server always use the default globalOS

	fifo_self = fifo_init();
	//Here, the getpid is the uuid used in local fifo
	global_fifo = global_fifo_init(getpid());

	fprintf(stderr, "Server global fifo:%d\n", global_fifo);

	//Let the server run forever
	while (true){
		ret = global_fifo_read(global_fifo, test_buf, MAX_TEST_BUF_SIZE);

		if (ret == -EFIFOLOCAL){
			ret = fifo_read(fifo_self, test_buf, MAX_TEST_BUF_SIZE);
			printf("receive:\n%s\n", test_buf);
		}
	}

	return EXIT_SUCCESS;
}
