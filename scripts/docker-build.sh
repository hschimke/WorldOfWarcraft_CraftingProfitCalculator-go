#! /bin/bash
# MUST BE RUN FROM ROOT OF REPOSITORY
readonly GIT_VERSION=`git rev-parse HEAD`
docker build -t cpc/web-serv -f docker/Dockerfile.web-serv --build-arg "GIT_VERSION=${GIT_VERSION}" .
docker build -t cpc/job-run -f docker/Dockerfile.job-run --build-arg "GIT_VERSION=${GIT_VERSION}" .
docker build -t cpc/injest -f docker/Dockerfile.injest --build-arg "GIT_VERSION=${GIT_VERSION}" .