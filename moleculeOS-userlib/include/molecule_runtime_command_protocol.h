/*
 * Molecule Runtime Command PROTOCOL
 * 	Basic neogitaion between Molecule Worker and Runc
 * */
#ifndef MOLECULE_RUNTIME_COMMAND_PROTOCOL_H
#define MOLECULE_RUNTIME_COMMAND_PROTOCOL_H

#define COMMAND_REQ_FORMAT ""
#define COMMAND_RESP_FORMAT ""
#define MAXIMUM_CONTAINER_ID_SIZE 64

char* start_server();
void init_server(int* fifo_self_ptr, int* global_fifo_ptr);
char* receive_from_server(int fifo_self, int global_fifo);
#endif
