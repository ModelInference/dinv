#/!usr/bin/env Rscript
 uninstlat <-read.table("./govectorEval/unmodifiedLatency.dat", header=TRUE, sep=",", na.strings="NA", dec=".", strip.white=TRUE)
 instlat <-read.table("./govectorEval/instrumentedLatency.dat", header=TRUE, sep=",", na.strings="NA", dec=".", strip.white=TRUE)
 uninstband <-read.table("./govectorEval/unmodifiedBandwidth.dat", header=TRUE, sep=",", na.strings="NA", dec=".", strip.white=TRUE)
 instband <-read.table("./govectorEval/instrumentedBandwidth.dat", header=TRUE, sep=",", na.strings="NA", dec=".", strip.white=TRUE)


print("unmodified latency")
summary(uninstlat)

print("instrumented latency")
summary(instlat)

print("unmodified bandwidth")
summary(uninstband)

print("instrumented bandwidth")
summary(instband)

