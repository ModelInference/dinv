#!/bin/bash
sudo -E go install ../../../../wantonsolutions/dovid
#hg revert client/kvclientmain.go
hg revert server/kvservicemain.go
#dovid client/kvclientmain.go
dovid server/kvservicemain.go
