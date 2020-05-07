# Version Printer

This is a subset of `k8s.io/component-base` used for easy version printer.


# How to use it

After import this repo, add ldflags to your go compile args like `-X github.com/mYmNeo/${package}.${key}=${val}`. Available package
 and
 values are listed below.
 
 |Key|Package|Shell script|
 |----|----|----|
 |ReleaseName|version|your favor|
 |gitMajor|verflag|your favor|
 |gitMinor|verflag|your favor|
 |gitVersion|verflag|your favor|
 |gitCommit|verflag|git rev-parse "HEAD^{commit}"|
 |gitTreeState|verflag|clean: git_status=$(git status --porcelain 2>/dev/null) && [[ -z ${git_status} ]];|
 |buildDate|verflag|date -u +'%Y-%m-%dT%H:%M:%SZ'|