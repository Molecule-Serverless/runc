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

void init_server(int* fifo_self_ptr, int* global_fifo_ptr){
	int fifo_self;
	int global_fifo;
	register_self_global(GLOBAL_OS_PORT); //server always use the default globalOS

	fifo_self = fifo_init();
	//Here, the getpid is the uuid used in local fifo
	global_fifo = global_fifo_init(getpid());
	*fifo_self_ptr = fifo_self;
	*global_fifo_ptr = global_fifo;
}

char* receive_from_server(int fifo_self, int global_fifo){
	int ret;

	#define MAX_TEST_BUF_SIZE 2048
	char* test_buf = malloc(MAX_TEST_BUF_SIZE);
	ret = global_fifo_read(global_fifo, test_buf, MAX_TEST_BUF_SIZE);

	if (ret == -EFIFOLOCAL){
		ret = fifo_read(fifo_self, test_buf, MAX_TEST_BUF_SIZE);
		printf("receive:\n%s\n", test_buf);
		return test_buf;
	}
}



/* used for debugging */
char* start_server() {
	// The file pointer we will associate with the FIFO
	int fifo_self;
	int fifo_server;
	int global_fifo;
	int ret;
#define MAX_TEST_BUF_SIZE 2048
	char* test_buf = malloc(MAX_TEST_BUF_SIZE);
	
	register_self_global(GLOBAL_OS_PORT); //server always use the default globalOS

	fifo_self = fifo_init();
	//Here, the getpid is the uuid used in local fifo
	global_fifo = global_fifo_init(getpid());

	fprintf(stderr, "Server global fifo:%d\n", global_fifo);

	//Let the server run forever
	//TODO: this loop should be put into the Go code. C code is used to accept a single command and Go decide how to handle the command
	while (true){
		ret = global_fifo_read(global_fifo, test_buf, MAX_TEST_BUF_SIZE);

		if (ret == -EFIFOLOCAL){
			ret = fifo_read(fifo_self, test_buf, MAX_TEST_BUF_SIZE);
			printf("receive:\n%s\n", test_buf);
			return test_buf;
		}
	}

	// return EXIT_SUCCESS;
}
