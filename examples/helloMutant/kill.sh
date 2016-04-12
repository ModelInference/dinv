#!/bin/bash
kill `ps | pgrep server | awk '{print $1}'`
kill `ps | pgrep client | awk '{print $1}'`
