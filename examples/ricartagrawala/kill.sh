#!/bin/bash
sleep 10
kill `ps | pgrep ricart | awk '{print $1}'`
