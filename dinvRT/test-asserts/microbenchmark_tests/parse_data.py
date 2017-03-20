#!/usr/bin/env python

"""
parse_data.py

Created by Amanda Carbonari on 12/15/2016.
"""

import os
import sys
import re
import numpy as np
import matplotlib.pyplot as plt

results = [0,0,0,0,0] #10, 20, 30, 40, 50 ms

def getIndex(category):
	if category == "10ms":
		return 0
	elif category == "20ms":
		return 1
	elif category == "30ms":
		return 2
	elif category == "40ms":
		return 3
	elif category == "50ms":
		return 4
	else:
		print("error:" + category)

def parse_file(input_file, res):
	parse_string = "\[\d{1,2}\] Elapsed \(ms\): (\d+.\d+)"

	fin = open(input_file, "r")
	
	for line in fin:
		parsed = re.match(parse_string, line)

		if parsed:
			res.append(float(parsed.group(1)))

	fin.close()

def parse_folder(folder):
	folder = os.path.abspath(folder)

	if not os.path.isdir(folder):
		sys.exit() # Print error

	cat = folder.split("-")[1]

	for input_file in os.listdir(folder):
		res = []
		full_path = os.path.join(folder, input_file)
		if not os.path.isdir(full_path) and "benchmark" in input_file:
			parse_file(full_path, res)
			results[getIndex(cat)] = np.mean(res)

def output_graph():
	plt.plot([10, 20, 30, 40, 50], results)
	plt.ylabel("Time (ms)")
	plt.xlabel("Round Trip Time (ms)")
	plt.savefig("RoundTripTimeGraph.pdf")

def output_file(output_file):
	fout = open(output_file, "w")

	fout.write("10ms: %.6f\n" % results[0])
	fout.write("20ms: %.6f\n" % results[1])
	fout.write("30ms: %.6f\n" % results[2])
	fout.write("40ms: %.6f\n" % results[3])
	fout.write("50ms: %.6f\n" % results[4])

	fout.close()

if __name__ == '__main__':
	for folder in os.listdir(os.path.abspath(".")):
		full_path = os.path.join(os.path.abspath("."), folder)
		if os.path.isdir(full_path) and "results" in folder:
			parse_folder(folder)
	output_graph()
	output_file("RoundTripTimeTable.txt")
