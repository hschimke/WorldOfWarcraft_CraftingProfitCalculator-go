#! /bin/bash
# MUST BE RUN FROM ROOT OF REPOSITORY
docker build -t cpc/web-serv -f docker/Dockerfile.web-serv .
docker build -t cpc/job-run -f docker/Dockerfile.job-run .
docker build -t cpc/injest -f docker/Dockerfile.injest .