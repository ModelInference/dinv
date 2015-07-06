/*
Fibb.c computes the fibbonacci numbers from 0 up to the given input

Stewart Grant 
May 1 2015
*/
#include <stdio.h>
#include <stdlib.h>

unsigned int * fibb(int n);
int validArgs(int argc, char **argv);
void printArray(unsigned int *array, int size);


char* usage = "./fibb arg";

int main(int argc, char **argv){
	if(validArgs < 0){
		exit(0);
	}
	unsigned int n;
	unsigned int *fibbs;
	n = atoi(argv[1]);
	fibbs = fibb(n);
	printArray(fibbs,n);
}

unsigned int * fibb(int n){
	unsigned int * fibbs = (unsigned int*) malloc(sizeof(unsigned int) * n);
	if(n >=1 )
		fibbs[0] = 1;
	if(n >=2)
		fibbs[1] = 1;
	int i;
	i=2;
	while (i<n){
		fibbs[i] = fibbs[i-1] + fibbs[i-2];
		i++;
	}
	return fibbs;
}
	
	
void printArray(unsigned int *array, int size){
	int i;
	for(i=0;i<size-1;i++){
		printf("%u\n",array[i]);
	}
	printf("%u\n",array[size-1]);
}

int validArgs(int argc, char**argv){
	int err=0;
	if(argc != 2){
		printf("%s\n",usage);
		err--;
	}
	if(atoi(argv[1]) < 0){
		printf("input argument must be a positive integer\n");
		err--;
	}
	return err;
}
		
