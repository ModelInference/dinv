#!/bin/bash
kill `ps | pgrep ricart | awk '{print $1}'`
